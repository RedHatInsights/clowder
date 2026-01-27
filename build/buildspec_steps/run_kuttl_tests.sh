echo "Running KUTTL tests..."
set +e  # Don't fail immediately on test failure
bash build/run_kuttl.sh --report xml
TEST_RC=$?
set -e
mv "${ARTIFACTS_DIR}/kuttl-report.xml" "${ARTIFACTS_DIR}/junit-kuttl.xml" || true

echo "Collecting logs and metrics..."
for p in $(kubectl get pod -n clowder-system -o jsonpath='{.items[*].metadata.name}'); do
    kubectl logs "$p" -n clowder-system > "${ARTIFACTS_DIR}/${p}.log" || true
    kubectl logs "$p" -n clowder-system --previous=true > "${ARTIFACTS_DIR}/${p}-previous.log" || true
done
kubectl -n clowder-system get all -o wide > "${ARTIFACTS_DIR}/clowder-system-get-all.txt" || true
kubectl get events --all-namespaces --sort-by=.lastTimestamp > "${ARTIFACTS_DIR}/cluster-events.txt" || true

( kubectl port-forward svc/clowder-controller-manager-metrics-service-non-auth -n clowder-system 8080 >/dev/null 2>&1 & echo $! > pf.pid ) || true
sleep 5 || true
curl -fsS http://127.0.0.1:8080/metrics > "${ARTIFACTS_DIR}/clowder-metrics" || true
kill "$(cat pf.pid)" 2>/dev/null || true

exit "$TEST_RC"
