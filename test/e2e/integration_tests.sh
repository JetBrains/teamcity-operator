#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# Integration tests for teamcity-operator.
#
# Inspired by the provider-sql integration-test approach: every sample
# configuration under config/samples/v1beta1/ is applied inside its own
# Kind namespace, verified for the resources the operator should create,
# and then cleaned up.
#
# Environment variables that control behaviour:
#   KIND_CLUSTER_NAME  – Kind cluster name          (default: teamcity-operator-e2e)
#   IMG                – operator image tag          (default: jetbrains/teamcity-operator:e2e-test)
#   CERT_MANAGER_VERSION – cert-manager version      (default: v1.14.3)
#   skipcleanup=true   – keep the Kind cluster after the run
#   skipsetup=true     – skip cluster/operator setup (re-run tests only)
#   skipbuild=true     – skip docker build (image already loaded)
# ---------------------------------------------------------------------------
set -e

# ========================== Output helpers ==================================
BLU='\033[0;34m'
YLW='\033[0;33m'
GRN='\033[0;32m'
RED='\033[0;31m'
NOC='\033[0m'

echo_info()           { printf "\n${BLU}%s${NOC}" "$1"; }
echo_step()           { printf "\n${BLU}>>>>>>> %s${NOC}\n" "$1"; }
echo_step_completed() { printf " ${GRN}[✔]${NOC}\n"; }
echo_success()        { printf "\n${GRN}%s${NOC}\n" "$1"; }
echo_warn()           { printf "\n${YLW}%s${NOC}\n" "$1"; }
echo_error()          { printf "\n${RED}%s${NOC}\n" "$1"; exit 1; }

# ========================== Paths & config ==================================
projectdir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
samplesdir="${projectdir}/config/samples/v1beta1"

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-teamcity-operator-e2e}"
IMG="${IMG:-jetbrains/teamcity-operator:e2e-test}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.14.3}"
KUBECTL="${KUBECTL:-kubectl}"
KIND="${KIND:-kind}"
KUSTOMIZE="${projectdir}/bin/kustomize"
READY_TIMEOUT="${READY_TIMEOUT:-900}"
ZDU_TIMEOUT="${ZDU_TIMEOUT:-1800}"
DELETE_TIMEOUT="${DELETE_TIMEOUT:-300}"
ZDU_FROM_IMAGE="${ZDU_FROM_IMAGE:-jetbrains/teamcity-server:2026.2.1}"
ZDU_TO_IMAGE="${ZDU_TO_IMAGE:-jetbrains/teamcity-server:2026.2.2}"

# Base64 value of "jdbc:mysql://mysql.default:3306/teamcity" used in sample
# files. Replaced per-namespace so that in-namespace MySQL is reachable.
SAMPLE_DB_URL_B64="amRiYzpteXNxbDovL215c3FsLmRlZmF1bHQ6MzMwNi90ZWFtY2l0eQ=="

# ========================== Cleanup trap ====================================
if [ "${skipcleanup}" != "true" ]; then
  function cleanup {
    echo_step "Cleaning up – deleting Kind cluster '${KIND_CLUSTER_NAME}'"
    "${KIND}" delete cluster --name="${KIND_CLUSTER_NAME}" 2>/dev/null || true
  }
  trap cleanup EXIT
fi

# ========================== Prerequisites ===================================
check_prerequisites() {
  echo_step "checking prerequisites"
  for cmd in docker "${KIND}" "${KUBECTL}"; do
    if ! command -v "${cmd}" &>/dev/null; then
      echo_error "'${cmd}' is required but not found in PATH"
    fi
  done
  echo_step_completed
}

# ========================== Cluster setup ===================================
setup_cluster() {
  echo_step "creating Kind cluster '${KIND_CLUSTER_NAME}'"
  "${KIND}" create cluster --name="${KIND_CLUSTER_NAME}" --wait=5m
  echo_step_completed
}

setup_cert_manager() {
  echo_step "installing cert-manager ${CERT_MANAGER_VERSION}"
  "${KUBECTL}" apply -f \
    "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"

  echo_info "  waiting for cert-manager deployments"
  "${KUBECTL}" wait --for=condition=Available deployment/cert-manager \
    -n cert-manager --timeout=180s
  "${KUBECTL}" wait --for=condition=Available deployment/cert-manager-webhook \
    -n cert-manager --timeout=180s
  "${KUBECTL}" wait --for=condition=Available deployment/cert-manager-cainjector \
    -n cert-manager --timeout=180s

  # Give the cert-manager webhook a moment to become fully operational
  sleep 10
  echo_step_completed
}

build_and_load_operator() {
  echo_step "building operator image '${IMG}'"
  docker build -t "${IMG}" "${projectdir}"

  echo_step "loading image into Kind cluster"
  "${KIND}" load docker-image "${IMG}" --name="${KIND_CLUSTER_NAME}"
  echo_step_completed
}

deploy_operator() {
  echo_step "deploying operator into the cluster"

  # Ensure manifests and kustomize binary are up-to-date
  make -C "${projectdir}" manifests kustomize

  # Point kustomize at the freshly built image
  cd "${projectdir}/config/manager" && "${KUSTOMIZE}" edit set image "controller=${IMG}"
  cd "${projectdir}"

  # Apply all operator manifests (CRDs, RBAC, webhooks, manager, cert-manager cert)
  "${KUSTOMIZE}" build config/default | "${KUBECTL}" apply --server-side -f -

  # Patch the manager Deployment for Kind:
  #   – set imagePullPolicy=IfNotPresent (image was pre-loaded)
  #   – remove imagePullSecrets (no registry credentials in Kind)
  echo_info "  patching manager deployment for Kind"
  "${KUBECTL}" patch deployment teamcity-operator-controller-manager \
    -n teamcity-operator-system --type=json \
    -p='[{"op":"add","path":"/spec/template/spec/containers/1/imagePullPolicy","value":"IfNotPresent"}]' \
    2>/dev/null || true
  "${KUBECTL}" patch deployment teamcity-operator-controller-manager \
    -n teamcity-operator-system --type=json \
    -p='[{"op":"remove","path":"/spec/template/spec/imagePullSecrets"}]' \
    2>/dev/null || true

  echo_info "  waiting for operator rollout"
  "${KUBECTL}" rollout status deployment/teamcity-operator-controller-manager \
    -n teamcity-operator-system --timeout=180s

  # Wait for the cert-manager-issued webhook certificate
  echo_info "  waiting for webhook certificate"
  "${KUBECTL}" wait --for=condition=Ready \
    certificate/teamcity-operator-serving-cert \
    -n teamcity-operator-system --timeout=120s 2>/dev/null || sleep 15

  # Verify the admission webhook is actually accepting requests
  wait_for_webhook
  echo_step_completed
}

wait_for_webhook() {
  echo_info "  waiting for admission webhook to respond"
  local timeout=120
  local elapsed=0

  while [ ${elapsed} -lt ${timeout} ]; do
    local output
    output=$("${KUBECTL}" apply --dry-run=server \
      -f "${samplesdir}/_v1beta1_teamcity.yaml" 2>&1 || true)

    # If there are no connection-level errors the webhook is operational
    if ! echo "${output}" | grep -qiE \
        "connection refused|no endpoints available|Internal error|i/o timeout"; then
      echo_step_completed
      return
    fi

    sleep 5
    elapsed=$((elapsed + 5))
  done

  echo_warn "  webhook readiness timed out after ${timeout}s – proceeding anyway"
}

# ========================== Test helpers ====================================

create_namespace() {
  local ns="$1"

  # If a previous namespace is still terminating, wait for it
  local phase
  phase=$("${KUBECTL}" get namespace "${ns}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
  if [ "${phase}" == "Terminating" ]; then
    echo_info "  waiting for namespace ${ns} to finish terminating"
    while "${KUBECTL}" get namespace "${ns}" >/dev/null 2>&1; do
      sleep 3
    done
  fi

  "${KUBECTL}" create namespace "${ns}" 2>/dev/null || true
}

# Apply a single-document (or simple multi-doc) sample into a test namespace.
# Replaces every "namespace: default" with the target namespace.
apply_sample() {
  local file="$1"
  local ns="$2"
  sed "s/namespace: default/namespace: ${ns}/g" "${file}" \
    | "${KUBECTL}" apply -n "${ns}" -f -
}

# Apply a multi-document sample that embeds MySQL resources + a Secret whose
# connectionUrl contains "mysql.default".  Fixes both the namespace and the
# base64-encoded JDBC URL so that in-namespace DNS resolution works.
apply_sample_with_db() {
  local file="$1"
  local ns="$2"

  local new_url_b64
  new_url_b64=$(echo -n "jdbc:mysql://mysql.${ns}:3306/teamcity" | base64 | tr -d '\n')

  sed -e "s/namespace: default/namespace: ${ns}/g" \
      -e "s/${SAMPLE_DB_URL_B64}/${new_url_b64}/g" \
      "${file}" | "${KUBECTL}" apply -n "${ns}" -f -
}

# Wait until at least $expected StatefulSets exist in the namespace.
wait_for_statefulsets() {
  local ns="$1"
  local expected="${2:-1}"
  local timeout="${3:-${STS_TIMEOUT:-180}}"
  local elapsed=0

  echo_info "  waiting for ${expected} StatefulSet(s) in ${ns}"
  while true; do
    local count
    count=$("${KUBECTL}" get statefulset -n "${ns}" --no-headers 2>/dev/null \
      | wc -l | tr -d ' ')
    if [ "${count}" -ge "${expected}" ]; then
      echo_step_completed
      return
    fi
    if [ "${elapsed}" -ge "${timeout}" ]; then
      echo ""
      echo_warn "--- Debug info for namespace ${ns} ---"
      "${KUBECTL}" get teamcity -n "${ns}" -o yaml 2>/dev/null || true
      "${KUBECTL}" get events -n "${ns}" --sort-by='.lastTimestamp' 2>/dev/null \
        | tail -30 || true
      echo_error "timeout (${timeout}s) waiting for ${expected} StatefulSet(s) in ${ns} (found ${count})"
    fi
    sleep 5
    elapsed=$((elapsed + 5))
  done
}

dump_namespace_debug() {
  local ns="$1"
  echo_warn "--- Debug info for namespace ${ns} ---"
  "${KUBECTL}" get teamcity,statefulset,pod,configmap -n "${ns}" -o wide 2>/dev/null || true
  "${KUBECTL}" describe pods -n "${ns}" 2>/dev/null | tail -80 || true
  local pod
  for pod in $("${KUBECTL}" get pods -n "${ns}" -o name 2>/dev/null); do
    echo "--- ${pod} logs"
    "${KUBECTL}" logs "${pod}" -n "${ns}" --all-containers --tail=40 2>/dev/null || true
  done
  "${KUBECTL}" get events -n "${ns}" --sort-by='.lastTimestamp' 2>/dev/null | tail -30 || true
}

verify_teamcity_healthy() {
  local ns="$1"
  local pod="$2"
  local out
  if ! out=$("${KUBECTL}" get --raw \
      "/api/v1/namespaces/${ns}/pods/${pod}:8111/proxy/healthCheck/healthy" 2>&1); then
    dump_namespace_debug "${ns}"
    echo_error "pod/${pod}: /healthCheck/healthy failed: ${out}"
  fi
  echo_info "  pod/${pod} /healthCheck/healthy OK"
  echo_step_completed
}

wait_for_teamcity_ready() {
  local ns="$1"
  local sts="$2"
  local timeout="${3:-${READY_TIMEOUT}}"

  echo_info "  waiting up to ${timeout}s for statefulset/${sts} to be Ready"
  if ! "${KUBECTL}" rollout status "statefulset/${sts}" -n "${ns}" --timeout="${timeout}s"; then
    dump_namespace_debug "${ns}"
    echo_error "statefulset/${sts} in ${ns} not Ready within ${timeout}s"
  fi
  verify_teamcity_healthy "${ns}" "${sts}-0"
}

pod_ready() {
  "${KUBECTL}" get pod "$2" -n "$1" \
    -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null
}

wait_for_owned_objects_released() {
  local ns="$1"
  local uids="$2"
  local elapsed=0
  local owned
  [ -z "${uids}" ] && return 0

  while true; do
    owned=$("${KUBECTL}" get statefulset,service,ingress,serviceaccount,persistentvolumeclaim -n "${ns}" \
      -o jsonpath='{range .items[*]}{.kind}/{.metadata.name} {.metadata.deletionTimestamp} {.metadata.ownerReferences[*].uid}{"\n"}{end}' \
      2>/dev/null | awk -v uids="${uids}" '
        BEGIN { n = split(uids, u, " "); for (i = 1; i <= n; i++) owner[u[i]] = 1 }
        { for (i = 2; i <= NF; i++) if ($i in owner) { if ($2 !~ /^[0-9]{4}-/) print $1; break } }')
    if [ -z "${owned}" ]; then
      echo_info "  objects owned by the deleted TeamCity are gone or terminating"
      echo_step_completed
      return 0
    fi
    if [ "${elapsed}" -ge "${DELETE_TIMEOUT}" ]; then
      dump_namespace_debug "${ns}"
      echo_error "objects owned by the deleted TeamCity were not garbage collected in ${ns}: $(echo ${owned})"
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
}

wait_for_teamcity_state() {
  local ns="$1"
  local tc="$2"
  local expected="$3"
  local elapsed=0
  local state
  while true; do
    state=$("${KUBECTL}" get teamcity "${tc}" -n "${ns}" -o jsonpath='{.status.state}' 2>/dev/null || true)
    if [ "${state}" == "${expected}" ]; then
      echo_info "  teamcity/${tc} status.state=${state}"
      echo_step_completed
      return 0
    fi
    if [ "${elapsed}" -ge 120 ]; then
      dump_namespace_debug "${ns}"
      echo_error "teamcity/${tc} status.state='${state}', expected '${expected}' ($("${KUBECTL}" get teamcity "${tc}" -n "${ns}" -o jsonpath='{.status.message}' 2>/dev/null))"
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
}

statefulset_field() {
  "${KUBECTL}" get statefulset "$2" -n "$1" -o jsonpath="$3" 2>/dev/null || true
}

expect_rejected() {
  local ns="$1"
  local case_name="$2"
  local needle="$3"
  local out
  if out=$("${KUBECTL}" apply --dry-run=server -n "${ns}" -f - 2>&1); then
    echo_error "${case_name}: invalid TeamCity was accepted"
  fi
  if ! grep -Fq -- "${needle}" <<<"${out}"; then
    echo_error "${case_name}: rejected for an unexpected reason: ${out}"
  fi
  echo_info "  ${case_name}: rejected"
  echo_step_completed
}

minimal_teamcity() {
  local name="$1"
  local spec_extra="$2"
  local main_extra="$3"
  cat <<EOF
apiVersion: jetbrains.com/v1beta1
kind: TeamCity
metadata:
  name: ${name}
spec:
  image: jetbrains/teamcity-server
${spec_extra}
  mainNode:
    name: main-node
    spec:
      requests:
        cpu: "500m"
        memory: "1000Mi"
${main_extra}
  dataDirVolumeClaim:
    name: data
    volumeMount:
      name: data
      mountPath: /storage
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
EOF
}

# Assert that at least $expected instances of a resource type exist.
verify_resource_exists() {
  local ns="$1"
  local resource_type="$2"
  local expected="${3:-1}"

  local count
  count=$("${KUBECTL}" get "${resource_type}" -n "${ns}" --no-headers 2>/dev/null \
    | wc -l | tr -d ' ')
  if [ "${count}" -lt "${expected}" ]; then
    echo_error "expected >= ${expected} ${resource_type}(s) in ${ns}, found ${count}"
  fi
  echo_info "  ${resource_type}: ${count} (expected >= ${expected})"
  echo_step_completed
}

# Deploy a standalone MySQL instance into a namespace (used by examples that
# reference a database-properties Secret but don't embed MySQL in the YAML).
deploy_mysql() {
  local ns="$1"
  echo_info "  deploying MySQL into ${ns}"

  "${KUBECTL}" create configmap mysql-initdb-config -n "${ns}" \
    --from-literal=init.sql="CREATE DATABASE IF NOT EXISTS teamcity;" \
    2>/dev/null || true

  cat <<'EOF' | "${KUBECTL}" apply -n "${ns}" -f -
apiVersion: v1
kind: Service
metadata:
  name: mysql
spec:
  ports:
    - port: 3306
  selector:
    app: mysql
  clusterIP: None
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql
spec:
  selector:
    matchLabels:
      app: mysql
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: mysql
    spec:
      volumes:
        - name: mysql-initdb
          configMap:
            name: mysql-initdb-config
      containers:
        - image: mysql:8
          name: mysql
          env:
            - name: MYSQL_ROOT_PASSWORD
              value: password
          ports:
            - containerPort: 3306
              name: mysql
          volumeMounts:
            - name: mysql-initdb
              mountPath: /docker-entrypoint-initdb.d
EOF

  "${KUBECTL}" wait --for=condition=Available deployment/mysql \
    -n "${ns}" --timeout=180s
  echo_step_completed
}

# Create the database-properties Secret expected by some samples.
create_database_secret() {
  local ns="$1"
  "${KUBECTL}" create secret generic database-properties -n "${ns}" \
    --from-literal=connectionProperties.password=password \
    --from-literal=connectionProperties.user=root \
    --from-literal="connectionUrl=jdbc:mysql://mysql.${ns}:3306/teamcity" \
    2>/dev/null || true
}

# Assert that a named PVC exists.
verify_pvc_named() {
  local ns="$1"
  local name="$2"

  if ! "${KUBECTL}" get pvc "${name}" -n "${ns}" >/dev/null 2>&1; then
    echo_error "expected PVC ${name} in ${ns}"
  fi
  echo_info "  pvc/${name} present"
  echo_step_completed
}

# Assert that a named PVC does not exist.
verify_pvc_absent() {
  local ns="$1"
  local name="$2"

  if "${KUBECTL}" get pvc "${name}" -n "${ns}" >/dev/null 2>&1; then
    echo_error "PVC ${name} should not exist in ${ns}"
  fi
  echo_info "  pvc/${name} absent (as expected)"
  echo_step_completed
}

# Create the ConfigMaps and Secret referenced by the custom mounts sample.
create_custom_mount_sources() {
  local ns="$1"

  "${KUBECTL}" create configmap teamcity-extra-config -n "${ns}" \
    --from-literal=extra.properties="some.property=value" \
    2>/dev/null || true
  "${KUBECTL}" create configmap teamcity-node-config -n "${ns}" \
    --from-literal=node.properties="secondary.only=true" \
    2>/dev/null || true
  "${KUBECTL}" create secret generic git-key -n "${ns}" \
    --from-literal=id_rsa="not-a-real-key" \
    2>/dev/null || true
}

# Assert a StatefulSet defines a volume backed by the given ConfigMap or Secret.
verify_sts_volume_source() {
  local ns="$1"
  local sts="$2"
  local volume_name="$3"
  local source_field="$4"
  local expected="$5"

  local actual
  actual=$("${KUBECTL}" get statefulset "${sts}" -n "${ns}" \
    -o jsonpath="{.spec.template.spec.volumes[?(@.name==\"${volume_name}\")].${source_field}}" \
    2>/dev/null)
  if [ "${actual}" != "${expected}" ]; then
    echo_error "StatefulSet ${sts}: volume ${volume_name} ${source_field}='${actual}', expected '${expected}'"
  fi
  echo_info "  sts/${sts} volume ${volume_name} -> ${expected}"
  echo_step_completed
}

# Assert a StatefulSet does not define a volume with the given name.
verify_sts_volume_absent() {
  local ns="$1"
  local sts="$2"
  local volume_name="$3"

  local actual
  actual=$("${KUBECTL}" get statefulset "${sts}" -n "${ns}" \
    -o jsonpath="{.spec.template.spec.volumes[?(@.name==\"${volume_name}\")].name}" \
    2>/dev/null)
  if [ -n "${actual}" ]; then
    echo_error "StatefulSet ${sts}: volume ${volume_name} should not be defined"
  fi
  echo_info "  sts/${sts} volume ${volume_name} absent (as expected)"
  echo_step_completed
}

# Assert the TeamCity container mounts a volume at the expected path.
verify_sts_container_mount_path() {
  local ns="$1"
  local sts="$2"
  local volume_name="$3"
  local expected_path="$4"

  local actual
  actual=$("${KUBECTL}" get statefulset "${sts}" -n "${ns}" \
    -o jsonpath="{.spec.template.spec.containers[0].volumeMounts[?(@.name==\"${volume_name}\")].mountPath}" \
    2>/dev/null)
  if [ "${actual}" != "${expected_path}" ]; then
    echo_error "StatefulSet ${sts}: container mount ${volume_name} at '${actual}', expected '${expected_path}'"
  fi
  echo_info "  sts/${sts} container mounts ${volume_name} at ${expected_path}"
  echo_step_completed
}

# Assert the TeamCity container does not mount a volume.
verify_sts_container_mount_absent() {
  local ns="$1"
  local sts="$2"
  local volume_name="$3"

  local actual
  actual=$("${KUBECTL}" get statefulset "${sts}" -n "${ns}" \
    -o jsonpath="{.spec.template.spec.containers[0].volumeMounts[?(@.name==\"${volume_name}\")].name}" \
    2>/dev/null)
  if [ -n "${actual}" ]; then
    echo_error "StatefulSet ${sts}: container should not mount ${volume_name}"
  fi
  echo_info "  sts/${sts} container does not mount ${volume_name} (as expected)"
  echo_step_completed
}

# Assert StatefulSet mounts a PVC claim by volume name.
verify_sts_volume_claim() {
  local ns="$1"
  local sts="$2"
  local volume_name="$3"
  local expected_claim="$4"

  local claim
  claim=$("${KUBECTL}" get statefulset "${sts}" -n "${ns}" \
    -o jsonpath="{.spec.template.spec.volumes[?(@.name==\"${volume_name}\")].persistentVolumeClaim.claimName}" \
    2>/dev/null)
  if [ "${claim}" != "${expected_claim}" ]; then
    echo_error "StatefulSet ${sts}: volume ${volume_name} claimName='${claim}', expected '${expected_claim}'"
  fi
  echo_info "  sts/${sts} volume ${volume_name} -> ${expected_claim}"
  echo_step_completed
}

# Assert TEAMCITY_SERVER_OPTS on a StatefulSet contains a substring.
verify_sts_server_opts_contains() {
  local ns="$1"
  local sts="$2"
  local needle="$3"

  local opts
  opts=$("${KUBECTL}" get statefulset "${sts}" -n "${ns}" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="TEAMCITY_SERVER_OPTS")].value}' \
    2>/dev/null)
  if ! echo "${opts}" | grep -Fq -- "${needle}"; then
    echo_error "StatefulSet ${sts}: TEAMCITY_SERVER_OPTS missing '${needle}' (got: ${opts})"
  fi
  echo_info "  sts/${sts} TEAMCITY_SERVER_OPTS contains ${needle}"
  echo_step_completed
}

# Create an empty RWO PVC for adopt tests.
create_existing_node_data_pvc() {
  local ns="$1"
  local name="$2"

  cat <<EOF | "${KUBECTL}" apply -n "${ns}" -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${name}
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF
}

# Clean up a test namespace. Deletes TeamCity CRs first so the operator can
# run its finaliser; fails if that times out or owned objects are left behind.
cleanup_namespace() {
  local ns="$1"
  echo_step "cleaning up namespace ${ns}"

  local uids
  uids=$("${KUBECTL}" get teamcity -n "${ns}" -o jsonpath='{.items[*].metadata.uid}' 2>/dev/null || true)
  if ! "${KUBECTL}" delete teamcity --all -n "${ns}" --timeout="${DELETE_TIMEOUT}s"; then
    dump_namespace_debug "${ns}"
    echo_error "TeamCity deletion in ${ns} did not finish within ${DELETE_TIMEOUT}s"
  fi
  wait_for_owned_objects_released "${ns}" "${uids}"
  "${KUBECTL}" delete namespace "${ns}" --wait=false 2>/dev/null || true
}

# Wrapper: execute a test function, measure duration, print result.
run_test() {
  local test_fn="$1"
  echo_step "======= TESTING: ${test_fn} ======="
  local start
  start=$(date +%s)

  ${test_fn}

  local duration=$(( $(date +%s) - start ))
  echo_success "======= ${test_fn} PASSED (${duration}s) ======="
}

# ========================== Test functions ==================================
# Each function creates its own namespace, applies the sample, verifies the
# resources the operator should have created, and cleans up.
# ============================================================================

test_webhook_rejects_invalid_specs() {
  local ns="test-webhook-validation"
  create_namespace "${ns}"

  if ! out=$(minimal_teamcity valid "" "" | "${KUBECTL}" apply --dry-run=server -n "${ns}" -f - 2>&1); then
    echo_error "valid minimal TeamCity was rejected: ${out}"
  fi
  echo_info "  valid minimal TeamCity: accepted"
  echo_step_completed

  minimal_teamcity xmx "  xmxPercentage: -1" "" \
    | expect_rejected "${ns}" "negative xmxPercentage" "Xmx percentage cannot be set to 0 or lower"

  minimal_teamcity dup-volumes "  volumes:
    - name: extra
      emptyDir: {}
    - name: extra
      emptyDir: {}" "" \
    | expect_rejected "${ns}" "duplicate volume names" "teamcity.spec.volumes[1].name: collides with teamcity.spec.volumes[0].name"

  minimal_teamcity dup-mount-path "  volumes:
    - name: first
      emptyDir: {}
      mount:
        mountPath: /storage
" "" \
    | expect_rejected "${ns}" "volume mount path colliding with the data dir" \
      "teamcity.spec.volumes[0].mount.mountPath: collides with teamcity.spec.dataDirVolumeClaim.volumeMount.mountPath"

  minimal_teamcity node-data-path "  startupPropertiesConfig:
    teamcity.node.data.path: /mnt/other" "      nodeDataDirVolumeClaim:
        name: node-data
        volumeMount:
          name: node-data
          mountPath: /mnt/node-data
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 1Gi" \
    | expect_rejected "${ns}" "node data path set twice" "teamcity.node.data.path is set by nodeDataDirVolumeClaim"

  cleanup_namespace "${ns}"
}

test_with_service_name_recreate() {
  local ns="test-sts-recreate"
  local tc="teamcity-sample-with-svc-recreate"
  local svc="tc-sample-recreate-svc"
  local annotation="teamcity.jetbrains.com/allow-sts-recreate"
  create_namespace "${ns}"

  local sample
  sample=$(mktemp)
  sed -e "\|${annotation}|d" -e "/serviceName: ${svc}/d" \
    "${samplesdir}/_v1beta1_teamcity_with_service_name_recreate.yaml" > "${sample}"
  apply_sample "${sample}" "${ns}"
  rm -f "${sample}"

  wait_for_statefulsets "${ns}" 2
  wait_for_teamcity_ready "${ns}" node
  wait_for_teamcity_ready "${ns}" secondary-node
  wait_for_teamcity_state "${ns}" "${tc}" Success

  local main_uid secondary_uid pvc_uid
  main_uid=$(statefulset_field "${ns}" node '{.metadata.uid}')
  secondary_uid=$(statefulset_field "${ns}" secondary-node '{.metadata.uid}')
  pvc_uid=$("${KUBECTL}" get pvc teamcity-node-volume1 -n "${ns}" -o jsonpath='{.metadata.uid}')

  local patch out
  patch="[{\"op\":\"add\",\"path\":\"/spec/mainNode/spec/serviceName\",\"value\":\"${svc}\"},{\"op\":\"add\",\"path\":\"/spec/secondaryNodes/0/spec/serviceName\",\"value\":\"${svc}\"}]"

  if out=$("${KUBECTL}" patch teamcity "${tc}" -n "${ns}" --type=json -p "${patch}" 2>&1); then
    echo_error "serviceName change without ${annotation} was accepted"
  fi
  if ! grep -Fq "requires annotation ${annotation}" <<<"${out}"; then
    echo_error "serviceName change rejected for an unexpected reason: ${out}"
  fi
  echo_info "  serviceName change without the annotation: rejected"
  echo_step_completed

  "${KUBECTL}" annotate teamcity "${tc}" -n "${ns}" "${annotation}=true"
  if ! out=$("${KUBECTL}" patch teamcity "${tc}" -n "${ns}" --type=json -p "${patch}" 2>&1); then
    echo_error "serviceName change with ${annotation}=true was rejected: ${out}"
  fi
  if ! grep -Fq "will be recreated" <<<"${out}"; then
    echo_error "serviceName change did not warn about the recreate: ${out}"
  fi
  echo_info "  serviceName change with the annotation: accepted with a recreate warning"
  echo_step_completed

  local elapsed=0 sts uid name
  for sts in node secondary-node; do
    [ "${sts}" == "node" ] && uid="${main_uid}" || uid="${secondary_uid}"
    while true; do
      name=$(statefulset_field "${ns}" "${sts}" '{.spec.serviceName}')
      if [ "${name}" == "${svc}" ] && [ "$(statefulset_field "${ns}" "${sts}" '{.metadata.uid}')" != "${uid}" ]; then
        echo_info "  statefulset/${sts} recreated with serviceName ${svc}"
        echo_step_completed
        break
      fi
      if [ "${elapsed}" -ge "${READY_TIMEOUT}" ]; then
        dump_namespace_debug "${ns}"
        echo_error "statefulset/${sts} was not recreated with serviceName ${svc} within ${READY_TIMEOUT}s"
      fi
      sleep 3
      elapsed=$((elapsed + 3))
    done
  done

  wait_for_teamcity_ready "${ns}" node
  wait_for_teamcity_ready "${ns}" secondary-node
  wait_for_teamcity_state "${ns}" "${tc}" Success

  if [ "$("${KUBECTL}" get pvc teamcity-node-volume1 -n "${ns}" -o jsonpath='{.metadata.uid}')" != "${pvc_uid}" ]; then
    echo_error "data directory PVC was replaced during the StatefulSet recreate"
  fi
  echo_info "  data directory PVC kept across the recreate"
  echo_step_completed

  if ! "${KUBECTL}" get events -n "${ns}" --field-selector reason=StatefulSetRecreating -o name | grep -q .; then
    echo_error "no StatefulSetRecreating event was recorded"
  fi
  echo_info "  StatefulSetRecreating event recorded"
  echo_step_completed

  cleanup_namespace "${ns}"
}

# --- 1. Basic standalone TeamCity ------------------------------------------
test_basic() {
  local ns="test-basic"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1
  verify_resource_exists "${ns}" pvc 1

  wait_for_teamcity_ready "${ns}" main-node

  cleanup_namespace "${ns}"
}

# --- 2. TeamCity with external database ------------------------------------
test_with_database() {
  local ns="test-database"
  create_namespace "${ns}"

  apply_sample_with_db "${samplesdir}/_v1beta1_teamcity_with_database.yaml" "${ns}"

  # MySQL Deployment is embedded in the multi-doc YAML
  "${KUBECTL}" wait --for=condition=Available deployment/mysql \
    -n "${ns}" --timeout=180s 2>/dev/null || true
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1
  verify_resource_exists "${ns}" deployment 1   # MySQL
  verify_resource_exists "${ns}" pvc 2           # dataDirVolumeClaim + config PVC

  wait_for_teamcity_ready "${ns}" main-node

  cleanup_namespace "${ns}"
}

# --- 3. TeamCity with Ingress ----------------------------------------------
test_with_ingress() {
  local ns="test-ingress"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_ingress.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1
  verify_resource_exists "${ns}" service 1   # from serviceList
  verify_resource_exists "${ns}" ingress 1   # from ingressList
  verify_resource_exists "${ns}" pvc 2       # dataDirVolumeClaim + config PVC

  cleanup_namespace "${ns}"
}

# --- 4. TeamCity with init containers --------------------------------------
test_with_init_containers() {
  local ns="test-init-containers"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_init_containers.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1

  # Verify the init container appears in the StatefulSet pod spec
  local init_json
  init_json=$("${KUBECTL}" get statefulset -n "${ns}" \
    -o jsonpath='{.items[0].spec.template.spec.initContainers[*].name}' 2>/dev/null)
  if [[ "${init_json}" != *"init-myservice"* ]]; then
    echo_error "init container 'init-myservice' not found in StatefulSet"
  fi
  echo_info "  init container 'init-myservice' present"
  echo_step_completed

  cleanup_namespace "${ns}"
}

# --- 5. TeamCity with node selector ----------------------------------------
test_with_node_selector() {
  local ns="test-node-selector"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_node_selector.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1

  local selector
  selector=$("${KUBECTL}" get statefulset -n "${ns}" \
    -o jsonpath='{.items[0].spec.template.spec.nodeSelector}' 2>/dev/null)
  if [[ "${selector}" != *"linux"* ]]; then
    echo_error "nodeSelector not set correctly on StatefulSet"
  fi
  echo_info "  nodeSelector correctly applied"
  echo_step_completed

  cleanup_namespace "${ns}"
}

# --- 6. TeamCity with custom Service --------------------------------------
test_with_service() {
  local ns="test-service"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_service.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1
  verify_resource_exists "${ns}" service 1   # from serviceList (tc-sample-svc)
  verify_resource_exists "${ns}" pvc 2       # dataDirVolumeClaim + config PVC

  cleanup_namespace "${ns}"
}

# --- 7. TeamCity with ServiceAccount ---------------------------------------
test_with_serviceaccount() {
  local ns="test-serviceaccount"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_serviceaccount.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1

  # Verify the SA exists with the expected annotation
  local ann
  ann=$("${KUBECTL}" get serviceaccount teamcity-service-account -n "${ns}" \
    -o jsonpath='{.metadata.annotations}' 2>/dev/null || echo "")
  if [[ "${ann}" != *"eks.amazonaws.com/role-arn"* ]]; then
    echo_error "ServiceAccount missing expected annotation"
  fi
  echo_info "  ServiceAccount annotation verified"
  echo_step_completed

  cleanup_namespace "${ns}"
}

# --- 8. TeamCity with startup properties -----------------------------------
test_with_startup_properties() {
  local ns="test-startup-props"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_startup_properties.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1

  cleanup_namespace "${ns}"
}

# --- 9. TeamCity with pod affinity -----------------------------------------
test_with_affinity() {
  local ns="test-affinity"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_affinity.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1

  # Verify the affinity block is present (pod may not schedule on a
  # non-matching architecture, but the StatefulSet must still be created)
  local affinity
  affinity=$("${KUBECTL}" get statefulset -n "${ns}" \
    -o jsonpath='{.items[0].spec.template.spec.affinity}' 2>/dev/null)
  if [ -z "${affinity}" ] || [ "${affinity}" == "{}" ]; then
    echo_error "affinity block not found in StatefulSet"
  fi
  echo_info "  affinity correctly applied on StatefulSet"
  echo_step_completed

  cleanup_namespace "${ns}"
}

# --- 10. TeamCity with secondary node (responsibilities) -------------------
test_with_secondary_node() {
  local ns="test-secondary-node"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_secondary_node.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 2   # main + secondary

  verify_resource_exists "${ns}" statefulset 2

  cleanup_namespace "${ns}"
}

# --- 11. TeamCity with secondary node (read-only, no responsibilities) -----
test_with_secondary_node_read_only() {
  local ns="test-secondary-ro"
  create_namespace "${ns}"

  # This sample references a database-properties Secret without embedding
  # MySQL, so we provision both before applying.
  deploy_mysql "${ns}"
  create_database_secret "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_secondary_node_read_only.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 2   # main + secondary

  verify_resource_exists "${ns}" statefulset 2

  wait_for_teamcity_ready "${ns}" main-node
  wait_for_teamcity_ready "${ns}" secondary-node

  cleanup_namespace "${ns}"
}

# --- 12. TeamCity with zero-downtime upgrade (single node) -----------------
test_with_zero_downtime_upgrade() {
  local ns="test-zero-downtime"
  local tc="teamcity-sample-zero-downtime"
  local main="main-node"
  local replica="${main}-update-replica"
  local checkpoint="update-checkpoint-${tc}"
  create_namespace "${ns}"

  local sample
  sample=$(mktemp)
  sed "s|^  image: jetbrains/teamcity-server$|  image: ${ZDU_FROM_IMAGE}|" \
    "${samplesdir}/_v1beta1_teamcity_with_zero_downtime_upgrade.yaml" > "${sample}"
  if ! grep -q "^  image: ${ZDU_FROM_IMAGE}$" "${sample}"; then
    rm -f "${sample}"
    echo_error "could not pin the zero-downtime sample to ${ZDU_FROM_IMAGE}"
  fi
  apply_sample_with_db "${sample}" "${ns}"
  rm -f "${sample}"

  "${KUBECTL}" wait --for=condition=Available deployment/mysql \
    -n "${ns}" --timeout=300s
  wait_for_statefulsets "${ns}" 1
  wait_for_teamcity_ready "${ns}" "${main}"

  # Verify the zero-downtime annotation
  local ann
  ann=$("${KUBECTL}" get teamcity "${tc}" -n "${ns}" \
    -o jsonpath='{.metadata.annotations}' 2>/dev/null)
  if [[ "${ann}" != *"zero-downtime"* ]]; then
    echo_error "zero-downtime annotation not found on TeamCity CR"
  fi

  local old_uid
  old_uid=$("${KUBECTL}" get pod "${main}-0" -n "${ns}" -o jsonpath='{.metadata.uid}')

  echo_info "  upgrading ${tc} from ${ZDU_FROM_IMAGE} to ${ZDU_TO_IMAGE}"
  "${KUBECTL}" patch teamcity "${tc}" -n "${ns}" --type=merge \
    -p "{\"spec\":{\"image\":\"${ZDU_TO_IMAGE}\"}}"

  local stages="" replica_ready=false outages=0 start now stage ready sts_image
  start=$(date +%s)
  while true; do
    now=$(date +%s)
    if [ $((now - start)) -ge "${ZDU_TIMEOUT}" ]; then
      dump_namespace_debug "${ns}"
      echo_error "zero-downtime upgrade did not finish within ${ZDU_TIMEOUT}s (stages seen:${stages:- none})"
    fi

    stage=$("${KUBECTL}" get configmap "${checkpoint}" -n "${ns}" \
      -o jsonpath='{.data.stage}' 2>/dev/null || true)
    if [ -n "${stage}" ] && [[ " ${stages} " != *" ${stage} "* ]]; then
      stages="${stages} ${stage}"
      echo_info "  [$((now - start))s] stage ${stage}"
    fi

    if [ "${replica_ready}" == "false" ] && [ "$(pod_ready "${ns}" "${replica}-0")" == "True" ]; then
      replica_ready=true
      echo_info "  [$((now - start))s] ${replica}-0 Ready"
    fi

    if ready=$("${KUBECTL}" get pods -n "${ns}" -l "app.kubernetes.io/name=${tc}" \
        -o jsonpath='{range .items[*]}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}' 2>/dev/null); then
      if ! echo "${ready}" | grep -q True; then
        outages=$((outages + 1))
        echo_warn "  [$((now - start))s] no Ready TeamCity pod"
      fi
    fi

    sts_image=$("${KUBECTL}" get statefulset "${main}" -n "${ns}" \
      -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true)
    if [ -z "${stage}" ] && [ "${sts_image}" == "${ZDU_TO_IMAGE}" ] \
        && ! "${KUBECTL}" get statefulset "${replica}" -n "${ns}" >/dev/null 2>&1 \
        && "${KUBECTL}" rollout status "statefulset/${main}" -n "${ns}" --timeout=1s >/dev/null 2>&1; then
      break
    fi
    sleep 3
  done
  echo_info "  upgrade finished after $(( $(date +%s) - start ))s, stages seen:${stages:- none}"
  echo_step_completed

  if [ "${replica_ready}" != "true" ]; then
    dump_namespace_debug "${ns}"
    echo_error "update replica ${replica}-0 never became Ready, so the upgrade was not zero-downtime"
  fi
  if [ "${outages}" -gt 0 ]; then
    dump_namespace_debug "${ns}"
    echo_error "no TeamCity pod was Ready in ${outages} sample(s) during the upgrade"
  fi

  local new_uid pod_image
  new_uid=$("${KUBECTL}" get pod "${main}-0" -n "${ns}" -o jsonpath='{.metadata.uid}')
  pod_image=$("${KUBECTL}" get pod "${main}-0" -n "${ns}" -o jsonpath='{.spec.containers[0].image}')
  if [ "${new_uid}" == "${old_uid}" ] || [ "${pod_image}" != "${ZDU_TO_IMAGE}" ]; then
    echo_error "pod/${main}-0 was not replaced with ${ZDU_TO_IMAGE} (image ${pod_image})"
  fi
  echo_info "  pod/${main}-0 replaced and running ${pod_image}"
  echo_step_completed
  verify_teamcity_healthy "${ns}" "${main}-0"

  echo_info "  starting a second upgrade and deleting ${tc} while it runs"
  "${KUBECTL}" patch teamcity "${tc}" -n "${ns}" --type=merge \
    -p "{\"spec\":{\"image\":\"${ZDU_FROM_IMAGE}\"}}"
  local waited=0
  until "${KUBECTL}" get configmap "${checkpoint}" -n "${ns}" >/dev/null 2>&1; do
    if [ "${waited}" -ge 120 ]; then
      dump_namespace_debug "${ns}"
      echo_error "checkpoint ${checkpoint} did not appear for the second upgrade"
    fi
    sleep 2
    waited=$((waited + 2))
  done
  local tc_uid
  tc_uid=$("${KUBECTL}" get teamcity "${tc}" -n "${ns}" -o jsonpath='{.metadata.uid}')
  if ! "${KUBECTL}" delete teamcity "${tc}" -n "${ns}" --timeout="${DELETE_TIMEOUT}s"; then
    dump_namespace_debug "${ns}"
    echo_error "deleting ${tc} during an upgrade did not finish within ${DELETE_TIMEOUT}s"
  fi
  if "${KUBECTL}" get configmap "${checkpoint}" -n "${ns}" >/dev/null 2>&1; then
    echo_error "checkpoint ${checkpoint} left behind after deleting ${tc} during an upgrade"
  fi
  echo_info "  finalizer removed the checkpoint of the interrupted upgrade"
  echo_step_completed
  wait_for_owned_objects_released "${ns}" "${tc_uid}"

  cleanup_namespace "${ns}"
}

# --- 13. TeamCity with secondary node + zero-downtime upgrade --------------
test_with_secondary_node_zero_downtime() {
  local ns="test-secondary-zdt"
  create_namespace "${ns}"

  apply_sample_with_db \
    "${samplesdir}/_v1beta1_teamcity_with_secondary_node_with_zero_downtime_upgrade.yaml" \
    "${ns}"

  "${KUBECTL}" wait --for=condition=Available deployment/mysql \
    -n "${ns}" --timeout=180s 2>/dev/null || true
  wait_for_statefulsets "${ns}" 2   # main + secondary

  verify_resource_exists "${ns}" statefulset 2

  cleanup_namespace "${ns}"
}

# --- 14. TeamCity with per-node data directory (greenfield) ----------------
test_with_node_data_dir() {
  local ns="test-node-data-dir"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_node_data_dir.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 2

  verify_resource_exists "${ns}" statefulset 2
  verify_resource_exists "${ns}" pvc 3

  verify_pvc_named "${ns}" "teamcity-data-dir"
  verify_pvc_named "${ns}" "node-data-dir-main-node"
  verify_pvc_named "${ns}" "node-data-dir-secondary-node"

  verify_sts_volume_claim "${ns}" "main-node" "node-data-dir" "node-data-dir-main-node"
  verify_sts_volume_claim "${ns}" "secondary-node" "node-data-dir" "node-data-dir-secondary-node"

  verify_sts_server_opts_contains "${ns}" "main-node" \
    "-Dteamcity.node.data.path=/mnt/node-data-dir"
  verify_sts_server_opts_contains "${ns}" "secondary-node" \
    "-Dteamcity.node.data.path=/mnt/node-data-dir"

  cleanup_namespace "${ns}"
}

# --- 15. TeamCity adopting an existing node-data PVC -----------------------
test_with_node_data_dir_adopt() {
  local ns="test-node-data-dir-adopt"
  local existing_claim="existing-node-data-dir"
  create_namespace "${ns}"

  create_existing_node_data_pvc "${ns}" "${existing_claim}"
  apply_sample "${samplesdir}/_v1beta1_teamcity_with_node_data_dir_adopt.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1
  verify_resource_exists "${ns}" pvc 2

  verify_pvc_named "${ns}" "teamcity-data-dir"
  verify_pvc_named "${ns}" "${existing_claim}"
  verify_pvc_absent "${ns}" "node-data-dir-main-node"

  verify_sts_volume_claim "${ns}" "main-node" "node-data-dir" "${existing_claim}"
  verify_sts_server_opts_contains "${ns}" "main-node" \
    "-Dteamcity.node.data.path=/mnt/node-data-dir"

  cleanup_namespace "${ns}"
}

# --- 16. TeamCity with custom volumes and per-node PVCs --------------------
test_with_custom_mounts() {
  local ns="test-custom-mounts"
  create_namespace "${ns}"

  create_custom_mount_sources "${ns}"
  apply_sample "${samplesdir}/_v1beta1_teamcity_with_custom_mounts.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 2

  verify_resource_exists "${ns}" statefulset 2
  verify_resource_exists "${ns}" pvc 3

  verify_pvc_named "${ns}" "teamcity-data-dir"
  verify_pvc_named "${ns}" "git-cache-main-node"
  verify_pvc_named "${ns}" "git-cache-secondary-node"

  verify_sts_volume_claim "${ns}" "main-node" "git-cache" "git-cache-main-node"
  verify_sts_volume_claim "${ns}" "secondary-node" "git-cache" "git-cache-secondary-node"
  verify_sts_container_mount_path "${ns}" "main-node" "git-cache" "/mnt/git-cache"

  verify_sts_volume_source "${ns}" "main-node" "extra-config" "configMap.name" "teamcity-extra-config"
  verify_sts_container_mount_path "${ns}" "main-node" "extra-config" "/mnt/extra-config"

  verify_sts_volume_source "${ns}" "main-node" "git-key" "secret.secretName" "git-key"
  verify_sts_container_mount_absent "${ns}" "main-node" "git-key"

  verify_sts_volume_source "${ns}" "secondary-node" "nodeconfig" "configMap.name" "teamcity-node-config"
  verify_sts_container_mount_path "${ns}" "secondary-node" "nodeconfig" "/mnt/nodeconfig"
  verify_sts_volume_absent "${ns}" "main-node" "nodeconfig"

  cleanup_namespace "${ns}"
}

# --- 17. TeamCity adopting an existing per-node PVC ------------------------
test_with_node_volume_claims_adopt() {
  local ns="test-node-volume-claims-adopt"
  local existing_claim="existing-git-cache"
  create_namespace "${ns}"

  create_existing_node_data_pvc "${ns}" "${existing_claim}"
  apply_sample "${samplesdir}/_v1beta1_teamcity_with_node_volume_claims_adopt.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1
  verify_resource_exists "${ns}" pvc 3

  verify_pvc_named "${ns}" "teamcity-data-dir"
  verify_pvc_named "${ns}" "${existing_claim}"
  verify_pvc_named "${ns}" "build-logs-main-node"

  verify_sts_volume_claim "${ns}" "main-node" "git-cache" "${existing_claim}"
  verify_sts_volume_claim "${ns}" "main-node" "build-logs" "build-logs-main-node"
  verify_sts_container_mount_path "${ns}" "main-node" "git-cache" "/mnt/git-cache"

  cleanup_namespace "${ns}"
}

# --- 18. TeamCity with a custom stop wait time ------------------------------
test_with_stop_wait_time() {
  local ns="test-stop-wait-time"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_stop_wait_time.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1

  local stop_wait grace
  stop_wait=$("${KUBECTL}" get statefulset main-node -n "${ns}" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="TEAMCITY_STOP_WAIT_TIME")].value}' \
    2>/dev/null)
  grace=$("${KUBECTL}" get statefulset main-node -n "${ns}" \
    -o jsonpath='{.spec.template.spec.terminationGracePeriodSeconds}' 2>/dev/null)
  if [ "${stop_wait}" != "120" ] || [ "${grace}" != "150" ]; then
    echo_error "StatefulSet main-node: TEAMCITY_STOP_WAIT_TIME='${stop_wait}', terminationGracePeriodSeconds='${grace}', expected 120 and 150"
  fi
  echo_info "  sts/main-node TEAMCITY_STOP_WAIT_TIME=120, terminationGracePeriodSeconds=150"
  echo_step_completed

  cleanup_namespace "${ns}"
}

# --- 19. TeamCity with topology spread constraints -------------------------
test_with_topology_spread_constraints() {
  local ns="test-topology-spread"
  create_namespace "${ns}"

  apply_sample "${samplesdir}/_v1beta1_teamcity_with_topology_spread_constraints.yaml" "${ns}"
  wait_for_statefulsets "${ns}" 1

  verify_resource_exists "${ns}" statefulset 1

  local topology_key
  topology_key=$(statefulset_field "${ns}" main-node \
    '{.spec.template.spec.topologySpreadConstraints[0].topologyKey}')
  if [ "${topology_key}" != "kubernetes.io/hostname" ]; then
    echo_error "StatefulSet main-node: topologySpreadConstraints[0].topologyKey='${topology_key}', expected kubernetes.io/hostname"
  fi
  echo_info "  sts/main-node spreads on kubernetes.io/hostname"
  echo_step_completed

  cleanup_namespace "${ns}"
}

# ========================== Main ============================================

check_prerequisites

if [ "${skipsetup}" != "true" ]; then
  setup_cluster
  setup_cert_manager

  if [ "${skipbuild}" != "true" ]; then
    build_and_load_operator
  fi

  deploy_operator
fi

# Run every example configuration as an isolated test
ALL_TESTS="test_webhook_rejects_invalid_specs test_basic test_with_database test_with_ingress test_with_init_containers test_with_node_selector test_with_service test_with_serviceaccount test_with_startup_properties test_with_affinity test_with_secondary_node test_with_secondary_node_read_only test_with_zero_downtime_upgrade test_with_secondary_node_zero_downtime test_with_node_data_dir test_with_node_data_dir_adopt test_with_custom_mounts test_with_node_volume_claims_adopt test_with_stop_wait_time test_with_topology_spread_constraints test_with_service_name_recreate"
for t in ${TESTS:-${ALL_TESTS}}; do
  run_test "${t}"
done

echo_success "All integration tests passed!"
