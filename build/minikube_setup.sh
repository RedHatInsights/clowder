#!/bin/bash
set +e

# Script you can use to set up a local minikube cluster for testing
# It is assumed you have already run 'minikube start' and your kubectl context is using the minikube cluster

# GO is required for yq, check if go is installed
echo "*** Checking for 'go' ..."
if ! command -v go; then
    echo "***  Go bin not found in path ***"
    echo "Please install go:"
    echo "sudo dnf install golang"
    exit 1
fi

GO_BIN_PATH="$(go env GOPATH)/bin"

export PATH="$PATH:$GO_BIN_PATH"

echo "*** Checking for 'yq' ..."
if ! command -v yq; then
    echo "*** 'yq' executable not found in '$GO_BIN_PATH', installing it with:"
    echo "         GO111MODULE=on go get github.com/mikefarah/yq/v4"
    (cd & GO111MODULE=on go get github.com/mikefarah/yq/v4)
fi


VERSION=0.21.1
OPERATOR_NS=strimzi
TEST_NS=test
WATCH_NS=${OPERATOR_NS},${TEST_NS}

if ! test -f strimzi-${VERSION}.tar.gz; then
    echo "*** Downloading strimzi-${VERSION}.tar.gz ..."
    wget https://github.com/strimzi/strimzi-kafka-operator/releases/download/${VERSION}/strimzi-${VERSION}.tar.gz
fi

echo "*** Unpacking .tar.gz ..."
tar xzf strimzi-${VERSION}.tar.gz

echo "Setting namespaces (OPERATOR_NS: $OPERATOR_NS, TEST_NS: $TEST_NS) in strimzi configs ..."
cd strimzi-${VERSION}/install/cluster-operator
# Set namespace that operator runs in
sed -i "s/namespace: .*/namespace: ${OPERATOR_NS}/" *RoleBinding*.yaml
# Set namespaces that operator watches
yq eval -i "del(.spec.template.spec.containers[0].env.[] | select(.name == \"STRIMZI_NAMESPACE\").valueFrom)" 060-Deployment-strimzi-cluster-operator.yaml
yq eval -i "(.spec.template.spec.containers[0].env.[] | select(.name == \"STRIMZI_NAMESPACE\")).value = \"$WATCH_NS\"" 060-Deployment-strimzi-cluster-operator.yaml

echo "*** Creating namespaces..."
kubectl create namespace $OPERATOR_NS || echo " ... ignoring that error"
kubectl create namespace $TEST_NS || echo " ... ignoring that error"

echo "*** Adding RoleBindings to namespaces watched by Strimzi ..."
IFS=","
for ns in $WATCH_NS; do
    kubectl apply -f 020-RoleBinding-strimzi-cluster-operator.yaml -n $ns
    kubectl apply -f 031-RoleBinding-strimzi-cluster-operator-entity-operator-delegation.yaml -n $ns
    kubectl apply -f 032-RoleBinding-strimzi-cluster-operator-topic-operator-delegation.yaml -n $ns
done

echo "*** Installing Strimzi resources ..."
kubectl apply -f . -n $OPERATOR_NS

echo "*** Waiting for Strimzi operator to come up ..."
kubectl rollout status deployment/strimzi-cluster-operator -n $OPERATOR_NS
