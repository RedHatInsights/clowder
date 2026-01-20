      echo "Preparing cluster dependencies (operators, CRDs)..."
      export KUBECTL_CMD="kubectl"
      export PATH="$PWD/bin:$PATH"
      # This script installs: Strimzi, cert-manager, prometheus-operator, 
      # cyndi-operator, elasticsearch-operator, keda-operator, and required CRDs
      bash build/kube_setup.sh