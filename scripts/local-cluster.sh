#!/usr/bin/env bash
# Cluster bootstrap for the local TeamCity simulation.
#
# Tilt (https://tilt.dev) drives the whole environment, but it resolves its
# Kubernetes connection once at startup. This script therefore only prepares
# the bits that must exist beforehand: the vind (vCluster in Docker) cluster
# and the kubeconfig Tilt will use (`.tilt-kubeconfig`).
#
# Usage:
#
#   scripts/local-cluster.sh            # create the cluster and the kubeconfig
#   scripts/local-cluster.sh --delete   # tear the cluster down
#
# Once this script finishes, run `tilt up` with KUBECONFIG pointing to
# .tilt-kubeconfig:

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUBECONFIG_FILE="$ROOT_DIR/.tilt-kubeconfig"
VALUES_FILE="$ROOT_DIR/local-cluster/vind-values.yaml"
CLUSTER_NAME="${CLUSTER_NAME:-teamcity}"

delete=0
if [[ "${1:-}" == "--delete" ]]; then
  delete=1
elif [[ $# -gt 0 ]]; then
  echo "Usage: $0 [--delete]" >&2
  exit 1
fi

require_tool() {
  if ! command -v "$1" &>/dev/null; then
    echo "error: $1 is required but not installed" >&2
    exit 1
  fi
}

require_tool docker
require_tool vcluster

cluster_running() {
  vcluster list --driver docker 2>/dev/null |
    awk -v name="$CLUSTER_NAME" '$1 == name {print $2}' |
    grep -qx running
}

if [[ $delete -eq 1 ]]; then
  vcluster delete "$CLUSTER_NAME" --driver docker
  rm -f "$KUBECONFIG_FILE"
  exit 0
fi

require_tool tilt

if ! docker info &>/dev/null; then
  echo "error: the Docker daemon is not running" >&2
  exit 1
fi

if ! cluster_running; then
  echo "==> Creating vind cluster '$CLUSTER_NAME' (a multi-node vCluster in Docker)..."
  vcluster create "$CLUSTER_NAME" \
    --driver docker \
    --values "$VALUES_FILE" \
    --connect=false
else
  echo "==> Reusing existing vind cluster '$CLUSTER_NAME'."
fi

echo "==> Generating $KUBECONFIG_FILE..."
vcluster connect "$CLUSTER_NAME" --driver docker --print >"$KUBECONFIG_FILE"

if ! grep -qE '^[^#].*\bteamcity\.local\b' /etc/hosts; then
  echo "==> WARNING: add this line to /etc/hosts, or http://teamcity.local:8111 will not resolve:"
  echo "    127.0.0.1 teamcity.local"
fi

cat <<EOF
==> Cluster '$CLUSTER_NAME' is ready.

Start the environment with:

  KUBECONFIG=$KUBECONFIG_FILE tilt up

Teardown:

  $0 --delete
EOF