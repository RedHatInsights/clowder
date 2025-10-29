#!/usr/bin/env bash
set -euo pipefail

# ---- helpers ----
diag() {
  echo "--- Diagnostics for namespace: $TEST_NS ---"
  echo "# oc get deploy -n $TEST_NS"
  oc get deploy -n "$TEST_NS" -o wide || true
  echo "# oc get pods -n $TEST_NS"
  oc get pods -n "$TEST_NS" -o wide || true
  echo "# oc describe deployments"
  for d in $(oc get deploy -n "$TEST_NS" -o name 2>/dev/null || true); do oc -n "$TEST_NS" describe "$d" || true; done
  echo "# oc describe pods"
  for p in $(oc get pods -n "$TEST_NS" -o name 2>/dev/null || true); do oc -n "$TEST_NS" describe "$p" || true; done
  echo "# Recent events"
  oc get events -n "$TEST_NS" --sort-by=.lastTimestamp | tail -n 100 || true
  echo "--- End diagnostics ---"
}

trap 'echo "[deploy.sh] error detected"; diag' ERR

# Required environment variables:
# - OC_SERVER: OpenShift API URL (e.g., https://api.cluster:6443)
# - OC_TOKEN: OpenShift Bearer token
# - TEST_NS: Namespace to deploy into (default: clowder-e2e)
# - RESOURCES_PATH: Path inside the repository to the YAML with resources (default: ci/full-cicd/clowder-test-resources.yaml)
# Optional:
# - WAIT_TIMEOUT: timeout for waits (default: 5m)

TEST_NS=${TEST_NS:-clowder-e2e}
WAIT_TIMEOUT=${WAIT_TIMEOUT:-5m}
RESOURCES_PATH=${RESOURCES_PATH:-ci/full-cicd/resources/puptoo-test-resources.yaml}
: "${OC_SERVER:?OC_SERVER is required}"
: "${OC_TOKEN:?OC_TOKEN is required}"

# Login non-interactively
oc login "$OC_SERVER" --token="$OC_TOKEN" --insecure-skip-tls-verify=true 1>/dev/null

# Ensure namespace exists
oc get namespace "$TEST_NS" >/dev/null 2>&1 || oc create namespace "$TEST_NS"

# Obtain resources (local path only)
WORKDIR=$(mktemp -d)
RES_FILE="$WORKDIR/resources.yaml"
if [[ -f "$RESOURCES_PATH" ]]; then
  cp "$RESOURCES_PATH" "$RES_FILE"
else
  echo "RESOURCES_PATH file not found: $RESOURCES_PATH" >&2
  exit 1
fi

# Simple placeholder substitution (CHANGE_ME_NS, CHANGE_ME_ENV)
PATCHED_FILE="$WORKDIR/resources.patched.yaml"
sed -e "s/CHANGE_ME_NS/$TEST_NS/g" "$RES_FILE" > "$PATCHED_FILE"
RES_FILE="$PATCHED_FILE"

echo "Applying test resources to namespace: $TEST_NS"
# If the YAML lacks namespace fields, use -n to apply; for namespaced objects this sets metadata.namespace.
oc apply -n "$TEST_NS" -f "$RES_FILE"

# Skip ClowdEnvironment readiness checks; focus on namespace workloads

# Wait for deployments rollout in the namespace (fail on timeout)
echo "Waiting for Deployments in namespace to be available..."
mapfile -t DEPLOYS < <(oc get deploy -n "$TEST_NS" -o name 2>/dev/null || true)
for d in "${DEPLOYS[@]:-}"; do
  [[ -n "$d" ]] || continue
  if ! oc -n "$TEST_NS" rollout status "$d" --timeout="$WAIT_TIMEOUT"; then
    echo "Deployment rollout failed or timed out: $d"
    diag
    exit 1
  fi
done

# Additionally, wait for pods to be Ready
echo "Waiting for Pods to be Ready..."
mapfile -t PODS < <(oc get pods -n "$TEST_NS" -o name 2>/dev/null || true)
for p in "${PODS[@]:-}"; do
  [[ -n "$p" ]] || continue
  if ! oc -n "$TEST_NS" wait --for=condition=Ready "$p" --timeout="$WAIT_TIMEOUT"; then
    echo "Pod not Ready in time: $p"
    diag
    exit 1
  fi
done

echo "Resources prepared at $RES_FILE"
# Persist for tests in a writable location
cp "$RES_FILE" /tmp/resources.yaml || true

echo "Done."
