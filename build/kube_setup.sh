#!/bin/bash
# Configures a local minikube cluster for testing.

set -e

REINSTALL=0

for arg in "$@"
do
    case $arg in
        -r|--reinstall)
        REINSTALL=1
        shift
        ;;
    esac
done

PLATFORM=`uname -a | cut -f1 -d' '`

# jq is required for cyndi operator install, check if jq is installed
echo "*** Checking for 'jq' ..."
if ! command -v jq; then
    echo "*** 'jq' not found in path. Please install jq with:"
    [[ $PLATFORM == "Darwin" ]] && echo "  brew install jq" \
        || echo "  sudo dnf install jq"
    exit 1
fi

# kubectl is required for interactions with the cluster.
if [ -n "${KUBECTL_CMD}" ]; then
    :  # already set via env var
elif command -v minikube; then
    KUBECTL_CMD='minikube kubectl --'
elif command -v kubectl; then
    KUBECTL_CMD=kubectl
else
    echo "*** 'kubectl' not found in path. Please install it or minikube, or set KUBECTL_CMD"
    exit 1
fi

if [ "$VIRTUAL_ENV" = "skip" ]; then
    echo "*** Skipping PyYAML installation (already provided by system)..."
else
    python3 -m venv "build/.build_venv"
    source build/.build_venv/bin/activate
    pip install --upgrade pip setuptools wheel
    pip install pyyaml
fi

declare -a BG_PIDS=()

ROOT_DIR=$(pwd)
DOWNLOAD_DIR="build/operator_bundles"
mkdir -p "$DOWNLOAD_DIR"


function install_strimzi_operator {
    STRIMZI_VERSION=0.45.1
    STRIMZI_OPERATOR_NS=strimzi
    WATCH_NS="*"
    STRIMZI_TARFILE="strimzi-${STRIMZI_VERSION}.tar.gz"
    FIX_NAMESPACE_SCRIPT="fix_namespace.py"

    if [ $REINSTALL -ne 1 ]; then
        STRIMZI_DEPLOYMENT=$(${KUBECTL_CMD} get deployment strimzi-cluster-operator -n $STRIMZI_OPERATOR_NS --ignore-not-found -o jsonpath='{.metadata.name}')
        if [ ! -z "$STRIMZI_DEPLOYMENT" ]; then
            echo "*** Strimzi operator deployment found, skipping install ..."
            return 0
        fi
    fi

    echo "*** Installing strimzi operator ..."
    cd "$DOWNLOAD_DIR"

    if ! test -f ${STRIMZI_TARFILE}; then
        echo "*** Downloading ${STRIMZI_TARFILE} ..."
        curl -LsSO https://github.com/strimzi/strimzi-kafka-operator/releases/download/${STRIMZI_VERSION}/${STRIMZI_TARFILE}
    fi

    echo "*** Unpacking .tar.gz ..."
    tar xzf ${STRIMZI_TARFILE}

    echo "Setting namespaces (STRIMZI_OPERATOR_NS: $STRIMZI_OPERATOR_NS, WATCH_NS: $WATCH_NS) in strimzi configs ..."
    cd strimzi-${STRIMZI_VERSION}/install/cluster-operator
    # Set namespace that operator runs in
    [[ $PLATFORM == "Darwin" ]] && sed -i '' "s/namespace: .*/namespace: ${STRIMZI_OPERATOR_NS}/" *RoleBinding*.yaml \
        || sed -i "s/namespace: .*/namespace: ${STRIMZI_OPERATOR_NS}/" *RoleBinding*.yaml

    [[ $PLATFORM == "Darwin" ]] && sed -i '' "s/memory: 384Mi/memory: 768Mi/" *Deployment*.yaml \
        || sed -i "s/memory: 384Mi/memory: 768Mi/" *Deployment*.yaml

    echo "*** Downloading ${FIX_NAMESPACE_SCRIPT} ..."
    curl -LsSO https://raw.githubusercontent.com/RedHatInsights/clowder/master/build/${FIX_NAMESPACE_SCRIPT} \
        -o ${FIX_NAMESPACE_SCRIPT} && chmod +x ${FIX_NAMESPACE_SCRIPT}
    mv ${FIX_NAMESPACE_SCRIPT} $ROOT_DIR/build/

    $ROOT_DIR/build/${FIX_NAMESPACE_SCRIPT} 060-Deployment-strimzi-cluster-operator.yaml "$WATCH_NS"

    echo "*** Creating ns ${STRIMZI_OPERATOR_NS}..."
    # if we hit an error, assumption is the Namespace already exists
    ${KUBECTL_CMD} create namespace $STRIMZI_OPERATOR_NS || echo " ... ignoring that error"

    echo "*** Adding cluster-wide RoleBindings for Strimzi ..."
    # if we hit an error, assumption is the ClusterRoleBinding already exists
    ${KUBECTL_CMD} create clusterrolebinding strimzi-cluster-operator-namespaced \
        --clusterrole=strimzi-cluster-operator-namespaced --serviceaccount ${STRIMZI_OPERATOR_NS}:strimzi-cluster-operator || echo " ... ignoring that error"
    ${KUBECTL_CMD} create clusterrolebinding strimzi-cluster-operator-entity-operator-delegation \
        --clusterrole=strimzi-entity-operator --serviceaccount ${STRIMZI_OPERATOR_NS}:strimzi-cluster-operator || echo " ... ignoring that error"
    ${KUBECTL_CMD} create clusterrolebinding strimzi-cluster-operator-watched \
        --clusterrole=strimzi-cluster-operator-watched --serviceaccount ${STRIMZI_OPERATOR_NS}:strimzi-cluster-operator || echo " ... ignoring that error"

    if [ $REINSTALL -ne 1 ]; then
        echo "*** Installing Strimzi resources ..."
        ${KUBECTL_CMD} create -f . -n $STRIMZI_OPERATOR_NS
    else
        echo "*** Replacing Strimzi resources ..."
        ${KUBECTL_CMD} replace -f . -n $STRIMZI_OPERATOR_NS
    fi

    echo "*** Will wait for Strimzi operator to come up in background"
    ${KUBECTL_CMD} rollout status deployment/strimzi-cluster-operator -n $STRIMZI_OPERATOR_NS | sed "s/^/[strimzi] /" &
    BG_PIDS+=($!)

    cd "$ROOT_DIR"
}

function install_cert_manager {
    CERT_MANAGER_VERSION=v1.5.3

    echo "*** Installing cert manager ..."
    cd "$DOWNLOAD_DIR"

    echo "*** Downloading ${CERT_MANAGER_YAML} ..."
    curl -LsSO https://github.com/jetstack/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml

    echo "*** Installing Cert Manager resources ..."
    ${KUBECTL_CMD} apply -f cert-manager.yaml

    echo "*** Will wait for cert manager to come up in background"
    ${KUBECTL_CMD} rollout status deployment/cert-manager -n cert-manager | sed "s/^/[cert-manager] /" &
    ${KUBECTL_CMD} rollout status deployment/cert-manager-webhook -n cert-manager | sed "s/^/[cert-manager] /" &
    BG_PIDS+=($!)

    cd "$ROOT_DIR"
}

function install_prometheus_operator {
    PROM_VERSION=0.56.3
    PROM_OPERATOR_NS=default
    PROM_TARFILE="prometheus-operator-${PROM_VERSION}.tar.gz"

    if [ $REINSTALL -ne 1 ]; then
        PROM_DEPLOYMENT=$(${KUBECTL_CMD} get deployment prometheus-operator -n $PROM_OPERATOR_NS --ignore-not-found -o jsonpath='{.metadata.name}')
        if [ ! -z "$PROM_DEPLOYMENT" ]; then
            echo "*** Prometheus operator deployment found, skipping install ..."
            return 0
        fi
    fi

    echo "*** Installing prometheus operator ..."
    cd "$DOWNLOAD_DIR"

    if ! test -f ${PROM_TARFILE}; then
        echo "*** Downloading ${PROM_TARFILE} ..."
        curl -LsS -o ${PROM_TARFILE} https://github.com/prometheus-operator/prometheus-operator/archive/v${PROM_VERSION}.tar.gz
    fi

    echo "*** Unpacking .tar.gz ..."
    tar xzf ${PROM_TARFILE}

    echo "*** Applying prometheus operator manifest ..."
    cd prometheus-operator-${PROM_VERSION}
    ${KUBECTL_CMD} create -f bundle.yaml --validate=false

    echo "*** Will wait for Prometheus operator to come up in background"
    ${KUBECTL_CMD} rollout status deployment/prometheus-operator -n $PROM_OPERATOR_NS | sed "s/^/[prometheus] /" &
    BG_PIDS+=($!)

    cd "$ROOT_DIR"
}

function install_cyndi_operator {
    OPERATOR_NS=cyndi-operator-system
    DEPLOYMENT=cyndi-operator-controller-manager

    if [ $REINSTALL -ne 1 ]; then
        OPERATOR_DEPLOYMENT=$(${KUBECTL_CMD} get deployment $DEPLOYMENT -n $OPERATOR_NS --ignore-not-found -o jsonpath='{.metadata.name}')
        if [ ! -z "$OPERATOR_DEPLOYMENT" ]; then
            echo "*** cyndi-operator deployment found, skipping install ..."
            return 0
        fi
    fi

    echo "*** Installing cyndi-operator ..."
    cd "$DOWNLOAD_DIR"

    echo "*** Looking up latest release ..."
    LATEST_MANIFEST=$(curl -sL https://api.github.com/repos/RedHatInsights/cyndi-operator/releases/latest | jq -r '.assets[].browser_download_url')
    echo "*** Downloading $LATEST_MANIFEST ..."
    curl -LsS $LATEST_MANIFEST -o cyndi-operator-manifest.yaml

    echo "*** Applying cyndi-operator manifest ..."
    ${KUBECTL_CMD} apply -f cyndi-operator-manifest.yaml

    echo "*** Will wait for cyndi-operator to come up in background"
    ${KUBECTL_CMD} rollout status deployment/$DEPLOYMENT -n $OPERATOR_NS | sed "s/^/[cyndi-operator] /" &
    BG_PIDS+=($!)

    cd "$ROOT_DIR"
}

function install_elasticsearch_operator {
    OPERATOR_NS=elastic-system
    POD=elastic-operator-0

    if [ $REINSTALL -ne 1 ]; then
        OPERATOR_POD=$(${KUBECTL_CMD} get pod $POD -n $OPERATOR_NS --ignore-not-found -o jsonpath='{.metadata.name}')
        if [ ! -z "$OPERATOR_POD" ]; then
            echo "*** elastic-operator-0 pod found, skipping install ..."
            return 0
        fi
    fi

    echo "*** Applying elastic-operator manifest ..."
    ${KUBECTL_CMD} create -f https://download.elastic.co/downloads/eck/2.2.0/crds.yaml
    ${KUBECTL_CMD} apply -f https://download.elastic.co/downloads/eck/2.2.0/operator.yaml

    echo "*** Will wait for elastic-operator to come up in background"
    ${KUBECTL_CMD} rollout status statefulset/elastic-operator -n "$OPERATOR_NS" | sed "s/^/[elastic-operator] /" &

    BG_PIDS+=($!)

    cd "$ROOT_DIR"
}

function install_keda_operator {
    OPERATOR_NS=keda
    DEPLOYMENT=keda-operator

    if [ $REINSTALL -ne 1 ]; then
        OPERATOR_DEPLOYMENT=$(${KUBECTL_CMD} get deployment $DEPLOYMENT -n $OPERATOR_NS --ignore-not-found -o jsonpath='{.metadata.name}')
        if [ ! -z "$OPERATOR_DEPLOYMENT" ]; then
            echo "*** keda-operator deployment found, skipping install ..."
            return 0
        fi
    fi

    echo "*** Applying keda-operator manifest ..."
    ${KUBECTL_CMD} apply -f https://github.com/kedacore/keda/releases/download/v2.12.0/keda-2.12.0.yaml --server-side

    echo "*** Will wait for keda-operator to come up in background"
    ${KUBECTL_CMD} rollout status deployment/$DEPLOYMENT -n $OPERATOR_NS | sed "s/^/[keda-operator] /" &
    BG_PIDS+=($!)

    cd "$ROOT_DIR"
}

function install_metrics_server {
    DEPLOYMENT=metrics-server
    OPERATOR_NS=kube-system

    # Check if metrics-server is already running
    if ${KUBECTL_CMD} get deployment $DEPLOYMENT -n $OPERATOR_NS &> /dev/null; then
        if ${KUBECTL_CMD} rollout status deployment/$DEPLOYMENT -n $OPERATOR_NS --timeout=5s &> /dev/null; then
            echo "*** metrics-server deployment found, skipping install ..."
            return
        fi
    fi

    echo "*** Installing metrics-server ..."
    ${KUBECTL_CMD} apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

    # Patch for Kind/local clusters that don't have valid kubelet certificates
    echo "*** Patching metrics-server for local clusters ..."
    ${KUBECTL_CMD} patch -n $OPERATOR_NS deployment $DEPLOYMENT --type=json \
        -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]' 2>/dev/null || true

    echo "*** Will wait for metrics-server to come up in background"
    ${KUBECTL_CMD} rollout status deployment/$DEPLOYMENT -n $OPERATOR_NS | sed "s/^/[metrics-server] /" &
    BG_PIDS+=($!)

    cd "$ROOT_DIR"
}

function install_subscription_crd {
    echo "*** Applying subscription CRD ..."
    ${KUBECTL_CMD} apply -f https://raw.githubusercontent.com/RedHatInsights/clowder/master/config/crd/static/subscriptions.operators.coreos.com.yaml

    cd "$ROOT_DIR"
}

function install_floorist_crd {
    echo "*** Applying Floorist CRD ..."
    ${KUBECTL_CMD} apply -k "https://github.com/RedHatInsights/floorist-operator/config/crd?ref=main"

    cd "$ROOT_DIR"
}

install_strimzi_operator
install_cert_manager
install_prometheus_operator
install_cyndi_operator
install_elasticsearch_operator
install_keda_operator
install_metrics_server
install_subscription_crd
install_floorist_crd

FAILURES=0
if [ ${#BG_PIDS[@]} -gt 0 ]; then
    echo "*** Waiting on background jobs to finish ..."
    for pid in ${BG_PIDS[*]}; do
        wait $pid || FAILURES+=1
    done
fi

if [ $FAILURES -gt 0 ]; then
    echo "*** ERROR: background job(s) failed"
    exit 1
else
    echo "*** Done!"
fi
