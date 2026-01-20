    echo "Deploying clowder operator and config..."
    kubectl create namespace clowder-system || true
    kubectl apply -f manifest.yaml --validate=false -n clowder-system
    kubectl apply -f clowder-config.yaml -n clowder-system
    kubectl delete pod -n clowder-system -l operator-name=clowder || true
    
    echo "=== Debugging clowder deployment before rollout ==="
    sleep 10
    kubectl get pods -n clowder-system -o wide
    kubectl get deployment -n clowder-system
    kubectl describe deployment clowder-controller-manager -n clowder-system || true
    kubectl get events -n clowder-system --sort-by='.lastTimestamp' | tail -20 || true

    CLOWDER_POD=$(kubectl get pod -n clowder-system -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [ -n "$CLOWDER_POD" ]; then
        echo "=== Clowder pod found: $CLOWDER_POD ==="
        kubectl describe pod "$CLOWDER_POD" -n clowder-system || true
        kubectl logs "$CLOWDER_POD" -n clowder-system --tail=50 || true
    else
        echo "=== No clowder pod found yet ==="
    fi
    
    echo "=== Starting rollout wait ==="
    kubectl rollout status deployment/clowder-controller-manager -n clowder-system --timeout=600s