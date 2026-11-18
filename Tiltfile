# Local multi-node TeamCity simulation, driven by Tilt on a vind cluster
# (vCluster in Docker). The cluster and .tilt-kubeconfig must already exist
# because Tilt resolves its Kubernetes connection once, at startup:
#
#   scripts/local-cluster.sh
#   KUBECONFIG=.tilt-kubeconfig tilt up
#
# TeamCity: http://teamcity.local:8111
#   (add '127.0.0.1 teamcity.local' to /etc/hosts if needed)
#
# Teardown: tilt down && scripts/local-cluster.sh --delete
#
# Resource graph:
#
#   cert-manager ───────────── teamcity-operator ──┐
#   traefik                                        ├─ teamcity
#   percona-postgresql-operator ─┐                 │
#   prerequisite-secrets ── teamcity-database ── teamcity-database-ready
#
# Everything is installed from Helm charts (preferring OCI registries), so
# Tilt only adds the bits Helm cannot express: building the operator image,
# applying the raw credentials and waiting for custom readiness conditions.

LOCAL_CLUSTER = 'local-cluster'

CLUSTER_NAME = os.getenv('CLUSTER_NAME', 'teamcity')
allow_k8s_contexts('vcluster-docker_' + CLUSTER_NAME)

# Cold-start chores (image pulls, first-time cert-manager webhook bootstraps)
# regularly take longer than the default 30s apply timeout.
update_settings(k8s_upsert_timeout_secs=900)

load('ext://helm_resource', 'helm_resource')

# The image is built into the host Docker daemon with a content-based tag.
# vind's pull-through registry proxy lets the cluster pull the image straight
# from there, so no push step is needed.
custom_build(
    'teamcity-operator',
    'docker build -t $EXPECTED_REF .',
    deps=[
        'Dockerfile',
        'go.mod',
        'go.sum',
        'cmd',
        'internal',
        'api',
    ],
    disable_push=True,
)

# cert-manager renders the webhook certificates used by the operator chart.
helm_resource(
    'cert-manager',
    chart='oci://quay.io/jetstack/charts/cert-manager',
    namespace='cert-manager',
    flags=[
        '--create-namespace',
        '--version', 'v1.17.2',
        '-f', LOCAL_CLUSTER + '/cert-manager-values.yaml',
    ],
    deps=[LOCAL_CLUSTER + '/cert-manager-values.yaml'],
    pod_readiness='wait',
)

# vind's automatic LoadBalancer forwarding needs root on macOS, so Traefik is
# served through the NodePort that vind publishes on port 8111 instead.
helm_resource(
    'traefik',
    chart='oci://ghcr.io/traefik/helm/traefik',
    namespace='traefik',
    flags=[
        '--create-namespace',
        '--version', '42.4.1',
        '-f', LOCAL_CLUSTER + '/traefik-values.yaml',
    ],
    deps=[LOCAL_CLUSTER + '/traefik-values.yaml'],
    pod_readiness='wait',
)

# The Percona PostgreSQL operator installs the CRDs and machinery that
# consume the PerconaPGCluster resources created by the database below.
helm_resource(
    'percona-postgresql-operator',
    chart='oci://ghcr.io/percona/percona-helm-charts/pg-operator',
    namespace='percona-postgresql-operator',
    flags=[
        '--create-namespace',
        '--version', '3.1.0',
        '-f', LOCAL_CLUSTER + '/pg-operator-values.yaml',
    ],
    deps=[LOCAL_CLUSTER + '/pg-operator-values.yaml'],
    pod_readiness='wait',
)

# Database credentials with fixed values, in plain YAML so they can be
# inspected and applied manually if needed.
local_resource(
    'prerequisite-secrets',
    'kubectl apply -f local-cluster/prerequisite-secrets.yaml',
    deps=[LOCAL_CLUSTER + '/prerequisite-secrets.yaml'],
)

helm_resource(
    'teamcity-database',
    chart='oci://ghcr.io/percona/percona-helm-charts/pg-db',
    namespace='teamcity',
    flags=[
        '--create-namespace',
        '--version', '3.1.0',
        '-f', LOCAL_CLUSTER + '/pg-db-values.yaml',
    ],
    deps=[LOCAL_CLUSTER + '/pg-db-values.yaml'],
    resource_deps=['percona-postgresql-operator', 'prerequisite-secrets'],
    pod_readiness='wait',
)

PG_READY_CMD = """
state=$(kubectl -n teamcity get perconapgcluster teamcity-database-pg -o jsonpath='{.status.state}')
waited=0
until [ "$state" = ready ]; do
  if [ "$waited" -ge 900 ]; then
    echo "The Postgres cluster did not become ready in time (state: $state)" >&2
    exit 1
  fi
  sleep 10
  waited=$((waited + 10))
  state=$(kubectl -n teamcity get perconapgcluster teamcity-database-pg -o jsonpath='{.status.state}')
done
echo 'The Postgres cluster is ready'
"""

# Helm returns once the Postgres resources exist, not once the cluster can
# serve connections, so its readiness is watched explicitly.
local_resource(
    'teamcity-database-ready',
    cmd=PG_READY_CMD,
    resource_deps=['teamcity-database'],
)

# Installing the operator chart is native Tilt+Helm territory: the chart is
# local, the image above is injected into it, and the webhook certificates
# come from cert-manager.
helm_resource(
    'teamcity-operator',
    chart='./charts/teamcity-operator',
    namespace='teamcity-operator',
    flags=['--create-namespace'],
    deps=['charts/teamcity-operator'],
    image_deps=['teamcity-operator'],
    image_keys=[
        ('controllerManager.manager.image.repository', 'controllerManager.manager.image.tag'),
    ],
    resource_deps=['cert-manager'],
    pod_readiness='wait',
)

# The TeamCity custom resource. It cannot be k8s_yaml because Tilt parses
# manifests against the cluster at load time, and the CRD only exists once
# the operator chart is installed.
TEAMCITY_CMD = """
set -e
kubectl wait --for=condition=Established crd/teamcities.jetbrains.com --timeout=15m
kubectl apply -f local-cluster/teamcity.yaml
# The operator takes a moment to create the StatefulSets after the CR lands.
waited=0
until kubectl -n teamcity get statefulset main-node >/dev/null 2>&1; do
  if [ "$waited" -ge 300 ]; then
    echo 'The operator did not create the main node StatefulSet in time' >&2
    exit 1
  fi
  sleep 5
  waited=$((waited + 5))
done
echo 'Waiting for the TeamCity main node (the first start takes a few minutes)...'
kubectl -n teamcity rollout status statefulset/main-node --timeout=15m
echo 'TeamCity is up: secondary nodes will join the main node shortly'
"""

local_resource(
    'teamcity',
    cmd=TEAMCITY_CMD,
    deps=[LOCAL_CLUSTER + '/teamcity.yaml'],
    resource_deps=['teamcity-operator', 'teamcity-database-ready'],
    links=['http://teamcity.local:8111'],
)
