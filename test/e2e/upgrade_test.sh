#!/usr/bin/env bash
set -euo pipefail

projectdir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
KUBECTL="${KUBECTL:-kubectl}"
OLD_REF="${OLD_REF:-0.0.27}"
OLD_IMG="${OLD_IMG:-jetbrains/teamcity-operator:${OLD_REF}}"
IMG="${IMG:-jetbrains/teamcity-operator:e2e-test}"
SETTLE_SECONDS="${SETTLE_SECONDS:-60}"
OPERATOR_NS="teamcity-operator-system"
DEPLOYMENT="teamcity-operator-controller-manager"
NS_PREFIX="upgrade"

workdir="$(mktemp -d)"
trap 'rm -rf "${workdir}"' EXIT

log()  { printf '\n>>> %s\n' "$1"; }
fail() { printf '\nFAIL: %s\n' "$1"; exit 1; }

render_manifests() {
  local src="$1" image="$2" out="$3"
  local name="${image%:*}" tag="${image##*:}"
  sed -i.bak -e "s#newName: .*#newName: ${name}#" -e "s#newTag: .*#newTag: ${tag}#" "${src}/config/manager/kustomization.yaml"
  "${KUBECTL}" kustomize "${src}/config/default" > "${out}"
}

deploy() {
  local manifests="$1" image="$2"
  "${KUBECTL}" apply --server-side --force-conflicts -f "${manifests}" >/dev/null
  "${KUBECTL}" -n "${OPERATOR_NS}" patch deployment "${DEPLOYMENT}" --type=json \
    -p='[{"op":"remove","path":"/spec/template/spec/imagePullSecrets"}]' >/dev/null 2>&1 || true
  "${KUBECTL}" -n "${OPERATOR_NS}" patch deployment "${DEPLOYMENT}" --type=strategic \
    -p="{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"manager\",\"imagePullPolicy\":\"IfNotPresent\"}]}}}}" >/dev/null
  "${KUBECTL}" -n "${OPERATOR_NS}" rollout status "deployment/${DEPLOYMENT}" --timeout=300s >/dev/null
  local running
  running="$("${KUBECTL}" -n "${OPERATOR_NS}" get deployment "${DEPLOYMENT}" -o jsonpath='{.spec.template.spec.containers[?(@.name=="manager")].image}')"
  [ "${running}" = "${image}" ] || fail "operator runs ${running}, expected ${image}"
  wait_for_webhook
}

wait_for_webhook() {
  local i
  for i in $(seq 1 60); do
    if "${KUBECTL}" apply --dry-run=server -n default -f "${workdir}/old/config/samples/v1beta1/_v1beta1_teamcity.yaml" >/dev/null 2>&1; then
      return 0
    fi
    sleep 3
  done
  fail "admission webhook did not become ready"
}

snapshot() {
  local ns="$1"
  {
    "${KUBECTL}" -n "${ns}" get statefulsets -o jsonpath='{range .items[*]}sts/{.metadata.name} uid={.metadata.uid} gen={.metadata.generation} rev={.status.updateRevision}{"\n"}{end}'
    "${KUBECTL}" -n "${ns}" get pods -o jsonpath='{range .items[*]}pod/{.metadata.name} uid={.metadata.uid}{"\n"}{end}'
    "${KUBECTL}" -n "${ns}" get pvc -o jsonpath='{range .items[*]}pvc/{.metadata.name} uid={.metadata.uid}{"\n"}{end}'
  } | sort
}

if ! git -C "${projectdir}" rev-parse -q --verify "refs/tags/${OLD_REF}^{commit}" >/dev/null; then
  git -C "${projectdir}" fetch -q --depth=1 "${OLD_REPO:-https://github.com/JetBrains/teamcity-operator.git}" "refs/tags/${OLD_REF}:refs/tags/${OLD_REF}" \
    || fail "cannot fetch ${OLD_REF}"
fi

mkdir -p "${workdir}/old" "${workdir}/new"
git -C "${projectdir}" archive "${OLD_REF}" config | tar -x -C "${workdir}/old"
cp -R "${projectdir}/config/." "${workdir}/new/config/"

render_manifests "${workdir}/old" "${OLD_IMG}" "${workdir}/old.yaml"
render_manifests "${workdir}/new" "${IMG}" "${workdir}/new.yaml"

log "installing operator ${OLD_IMG} from ${OLD_REF}"
deploy "${workdir}/old.yaml" "${OLD_IMG}"

log "applying ${OLD_REF} samples"
namespaces=()
for sample in "${workdir}"/old/config/samples/v1beta1/*.yaml; do
  base="$(basename "${sample}" .yaml)"
  ns="${NS_PREFIX}-$(printf '%s' "${base#_v1beta1_}" | tr '_' '-' | cut -c1-50)"
  "${KUBECTL}" create namespace "${ns}" >/dev/null 2>&1 || true
  "${KUBECTL}" -n "${ns}" create quota no-pods --hard=pods=0 >/dev/null 2>&1 || true
  if sed "s/namespace: default/namespace: ${ns}/g" "${sample}" | "${KUBECTL}" apply -n "${ns}" -f - >"${workdir}/${ns}.apply" 2>&1; then
    namespaces+=("${ns}")
  else
    printf '  skip %-60s %s\n' "${base}" "$(tail -1 "${workdir}/${ns}.apply" | cut -c1-120)"
  fi
done
[ "${#namespaces[@]}" -gt 0 ] || fail "no sample could be applied with ${OLD_REF}"

log "waiting ${SETTLE_SECONDS}s for ${OLD_REF} to settle"
sleep "${SETTLE_SECONDS}"
for ns in "${namespaces[@]}"; do snapshot "${ns}" > "${workdir}/${ns}.a"; done
sleep "${SETTLE_SECONDS}"
stable=()
for ns in "${namespaces[@]}"; do
  snapshot "${ns}" > "${workdir}/${ns}.b"
  if ! grep -q '^sts/' "${workdir}/${ns}.b"; then
    printf '  skip %-60s no StatefulSet under %s\n' "${ns}" "${OLD_REF}"
  elif diff -q "${workdir}/${ns}.a" "${workdir}/${ns}.b" >/dev/null; then
    stable+=("${ns}")
  else
    printf '  skip %-60s still changing under %s\n' "${ns}" "${OLD_REF}"
  fi
done
[ "${#stable[@]}" -gt 0 ] || fail "no sample settled under ${OLD_REF}"

log "upgrading operator to ${IMG}"
deploy "${workdir}/new.yaml" "${IMG}"
sleep "${SETTLE_SECONDS}"

log "checking that the upgrade left ${#stable[@]} TeamCity installations untouched"
failed=0
for ns in "${stable[@]}"; do
  snapshot "${ns}" > "${workdir}/${ns}.c"
  if diff -u "${workdir}/${ns}.b" "${workdir}/${ns}.c" > "${workdir}/${ns}.diff"; then
    printf '  ok   %s\n' "${ns}"
  else
    printf '  FAIL %s changed after the upgrade:\n' "${ns}"
    sed -n '3,$p' "${workdir}/${ns}.diff" | sed 's/^/         /'
    failed=1
  fi
done

log "checking that existing TeamCity resources still pass admission after the upgrade"
for ns in "${stable[@]}"; do
  if "${KUBECTL}" -n "${ns}" annotate teamcity --all e2e.jetbrains.com/upgrade-check="$(date +%s)" --overwrite >"${workdir}/${ns}.annotate" 2>&1; then
    printf '  ok   %s\n' "${ns}"
  else
    printf '  FAIL %s: %s\n' "${ns}" "$(tail -1 "${workdir}/${ns}.annotate")"
    failed=1
  fi
done

for ns in "${namespaces[@]}"; do "${KUBECTL}" delete namespace "${ns}" --wait=false >/dev/null 2>&1 || true; done
for ns in "${namespaces[@]}"; do "${KUBECTL}" wait --for=delete "namespace/${ns}" --timeout=300s >/dev/null 2>&1 || true; done

[ "${failed}" -eq 0 ] || fail "upgrade from ${OLD_REF} changed or broke existing installations"
printf '\nUpgrade from %s to %s passed for %d installations\n' "${OLD_REF}" "${IMG}" "${#stable[@]}"
