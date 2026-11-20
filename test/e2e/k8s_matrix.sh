#!/usr/bin/env bash
set -uo pipefail

projectdir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MATRIX="${MATRIX:-kind:v1.26.15 minikube:v1.30.14 minikube:v1.33.4 kind:v1.37.0}"
PARALLEL="${PARALLEL:-2}"
IMG="${IMG:-jetbrains/teamcity-operator:e2e-test}"
OLD_REF="${OLD_REF:-0.0.28}"
OLD_IMG="${OLD_IMG:-jetbrains/teamcity-operator:${OLD_REF}}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.14.3}"
MINIKUBE_CPUS="${MINIKUBE_CPUS:-4}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-6g}"
LOGDIR="${LOGDIR:-${TMPDIR:-/tmp}/teamcity-operator-e2e}"
KEEP_CLUSTERS="${KEEP_CLUSTERS:-false}"
STS_TIMEOUT="${STS_TIMEOUT:-600}"
PRELOAD_IMAGES="${PRELOAD_IMAGES:-jetbrains/teamcity-server:latest jetbrains/teamcity-server:2026.2 mysql:8 busybox:latest busybox:1.28 busybox:1.36}"

if [ -z "${LOAD_PULLED_IMAGES:-}" ]; then
  if docker info -f '{{range .DriverStatus}}{{index . 0}}={{index . 1}} {{end}}' 2>/dev/null | grep -q 'driver-type=io.containerd.snapshotter'; then
    LOAD_PULLED_IMAGES=false
  else
    LOAD_PULLED_IMAGES=true
  fi
fi
if [ "${LOAD_PULLED_IMAGES}" != "true" ]; then
  echo "Docker uses the containerd image store: clusters pull registry images themselves"
  PULLED_IMAGES=""
else
  PULLED_IMAGES="${OLD_IMG} ${PRELOAD_IMAGES}"
fi

mkdir -p "${LOGDIR}"
rm -f "${LOGDIR}"/*.result

cluster_name() { printf 'tco-e2e-%s-%s' "$1" "$(printf '%s' "${2%%@*}" | tr -d 'v' | tr '.' '-')"; }

create_cluster() {
  local provider="$1" version="$2" name="$3"
  case "${provider}" in
    kind)
      kind create cluster --name "${name}" --image "kindest/node:${version}" --wait 5m
      ;;
    minikube)
      minikube start -p "${name}" --driver=docker --kubernetes-version="${version}" \
        --cpus="${MINIKUBE_CPUS}" --memory="${MINIKUBE_MEMORY}" --wait=all
      ;;
    *) echo "unknown provider ${provider}"; return 1 ;;
  esac
}

load_image() {
  local provider="$1" name="$2" image="$3"
  case "${provider}" in
    kind) kind load docker-image "${image}" --name "${name}" ;;
    minikube) minikube -p "${name}" image load "${image}" ;;
  esac
}

load_images() {
  local provider="$1" name="$2" image
  for image in "${IMG}" ${PULLED_IMAGES}; do
    load_image "${provider}" "${name}" "${image}" || return 1
  done
}

delete_cluster() {
  local provider="$1" name="$2"
  case "${provider}" in
    kind) kind delete cluster --name "${name}" ;;
    minikube) minikube delete -p "${name}" ;;
  esac
}

install_cert_manager() {
  kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml" >/dev/null
  local d
  for d in cert-manager cert-manager-webhook cert-manager-cainjector; do
    kubectl -n cert-manager wait --for=condition=Available "deployment/${d}" --timeout=300s
  done
  sleep 10
}

run_one() {
  local provider="$1" version="$2"
  local name; name="$(cluster_name "${provider}" "${version}")"
  local log="${LOGDIR}/${name}.log"
  local started; started="$(date +%s)"
  local cluster="fail" upgrade="not run" integration="not run"
  export KUBECONFIG="${LOGDIR}/${name}.kubeconfig"
  rm -f "${KUBECONFIG}"
  {
    echo "=== ${provider} ${version} (${name})"
    if create_cluster "${provider}" "${version}" "${name}" && install_cert_manager \
      && load_images "${provider}" "${name}"; then
      cluster="ok"
      kubectl version 2>/dev/null | grep -i server
      if IMG="${IMG}" OLD_REF="${OLD_REF}" OLD_IMG="${OLD_IMG}" bash "${scriptdir}/upgrade_test.sh"; then
        upgrade="pass"
      else
        upgrade="FAIL"
      fi
      if skipsetup=true skipcleanup=true STS_TIMEOUT="${STS_TIMEOUT}" IMG="${IMG}" bash "${scriptdir}/integration_tests.sh"; then
        integration="pass"
      else
        integration="FAIL"
      fi
    fi
    if [ "${KEEP_CLUSTERS}" != "true" ]; then
      delete_cluster "${provider}" "${name}"
    fi
  } >"${log}" 2>&1
  printf '%s|%s|%s|%s|%s|%ss\n' "${provider}" "${version}" "${cluster}" "${upgrade}" "${integration}" "$(( $(date +%s) - started ))" \
    > "${LOGDIR}/${name}.result"
  echo "finished ${provider} ${version}: cluster=${cluster} upgrade=${upgrade} integration=${integration} (log: ${log})"
}

echo "building ${IMG}"
docker build -q -t "${IMG}" "${projectdir}" >/dev/null || { echo "image build failed"; exit 1; }
for image in ${PULLED_IMAGES}; do
  echo "pulling ${image}"
  docker pull -q "${image}" >/dev/null || { echo "cannot pull ${image}"; exit 1; }
done

running=0
for entry in ${MATRIX}; do
  run_one "${entry%%:*}" "${entry#*:}" &
  running=$((running + 1))
  if [ "${running}" -ge "${PARALLEL}" ]; then
    wait -n 2>/dev/null || wait
    running=$((running - 1))
  fi
done
wait

printf '\n%-9s %-9s %-8s %-8s %-12s %s\n' PROVIDER K8S CLUSTER UPGRADE INTEGRATION TIME
status=0
for f in "${LOGDIR}"/*.result; do
  IFS='|' read -r p v c u i t < "${f}"
  printf '%-9s %-9s %-8s %-8s %-12s %s\n' "${p}" "${v}" "${c}" "${u}" "${i}" "${t}"
  if [ "${c}" != "ok" ] || [ "${u}" != "pass" ] || [ "${i}" != "pass" ]; then status=1; fi
done
exit "${status}"
