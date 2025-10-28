#!/usr/bin/env bash
set -euo pipefail

# ---- helpers ----
diag() {
  echo "--- Diagnostics for namespace: $TEST_NS ---"
  echo "ClowdEnvironment (if any): ${CE_NAME:-<unknown>} (cluster-scoped)"
  oc get clowdenvironment -o wide || true
  if [[ -n "${CE_NAME:-}" ]]; then
    echo "# oc get clowdenvironment/$CE_NAME -o yaml"
    oc get clowdenvironment "$CE_NAME" -o yaml || true
    echo "# oc describe clowdenvironment/$CE_NAME"
    oc describe clowdenvironment "$CE_NAME" || true
  fi
  echo "# oc get all"
  oc get all -n "$TEST_NS" || true
  echo "# oc get clowdapp -n $TEST_NS"
  oc get clowdapp -n "$TEST_NS" -o wide || true
  echo "# oc describe deployments"
  for d in $(oc get deploy -n "$TEST_NS" -o name 2>/dev/null || true); do oc -n "$TEST_NS" describe "$d" || true; done
  echo "# Recent events"
  oc get events -n "$TEST_NS" --sort-by=.lastTimestamp | tail -n 100 || true
  echo "--- End diagnostics ---"
}

trap 'echo "[deploy.sh] error detected"; diag' ERR

# Required environment variables:
# - OC_SERVER: OpenShift API URL (e.g., https://api.cluster:6443)
# - OC_TOKEN: OpenShift Bearer token
# - TEST_NS: Namespace to deploy into (default: clowder-e2e)
# - RESOURCES_URL: HTTP(S) URL or local path to a YAML with ClowdEnv/ClowdApp resources
# Optional:
# - WAIT_TIMEOUT: timeout for waits (default: 5m)

TEST_NS=${TEST_NS:-clowder-e2e}
WAIT_TIMEOUT=${WAIT_TIMEOUT:-5m}
: "${OC_SERVER:?OC_SERVER is required}"
: "${OC_TOKEN:?OC_TOKEN is required}"
: "${RESOURCES_URL:?RESOURCES_URL is required}"

# Login non-interactively
oc login "$OC_SERVER" --token="$OC_TOKEN" --insecure-skip-tls-verify=true 1>/dev/null

# Ensure namespace exists
oc get namespace "$TEST_NS" >/dev/null 2>&1 || oc create namespace "$TEST_NS"

# Obtain resources
WORKDIR=$(mktemp -d)
RES_FILE="$WORKDIR/resources.yaml"
if [[ "$RESOURCES_URL" =~ ^https?:// ]]; then
  curl -sSL "$RESOURCES_URL" -o "$RES_FILE"
else
  if [[ -f "$RESOURCES_URL" ]]; then
    cp "$RESOURCES_URL" "$RES_FILE"
  else
    echo "RESOURCES_URL is neither a valid URL nor an existing file: $RESOURCES_URL" >&2
    exit 1
  fi
fi

# Simple placeholder substitution (CHANGE_ME_NS, CHANGE_ME_ENV)
PATCHED_FILE="$WORKDIR/resources.patched.yaml"
sed -e "s/CHANGE_ME_NS/$TEST_NS/g" "$RES_FILE" > "$PATCHED_FILE"
RES_FILE="$PATCHED_FILE"

echo "Applying test resources to namespace: $TEST_NS"
# If the YAML lacks namespace fields, use -n to apply; for namespaced objects this sets metadata.namespace.
oc apply -n "$TEST_NS" -f "$RES_FILE"

# Wait for ClowdEnvironment readiness (fail on timeout)
echo "Waiting for ClowdEnvironment to be Ready..."
CE_NAME=$(oc get -f "$RES_FILE" -o jsonpath='{range .items[?(@.kind=="ClowdEnvironment")]}{.metadata.name}{"\n"}{end}' 2>/dev/null | head -n1)
# Fallback if jsonpath detection fails
if [[ -z "$CE_NAME" ]]; then
  CE_NAME=$(grep -A2 -m1 '^kind: ClowdEnvironment' "$RES_FILE" | grep -m1 '^metadata:' -A2 | grep -m1 '^  name:' | awk '{print $2}' || true)
fi
if [[ -n "$CE_NAME" ]]; then
  echo "Found ClowdEnvironment: $CE_NAME"
  echo "Current CE status:"
  oc get clowdenvironment "$CE_NAME" -o jsonpath='{.status}' || true; echo
  if ! oc wait --for=jsonpath='{.status.ready}'=true clowdenvironment "$CE_NAME" --timeout="$WAIT_TIMEOUT"; then
    echo "ClowdEnvironment did not become Ready within $WAIT_TIMEOUT"
    diag
    exit 1
  fi
fi

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

echo "Resources prepared at $RES_FILE"
# Persist for tests
mkdir -p /workspace
cp "$RES_FILE" /workspace/resources.yaml

echo "Done."
