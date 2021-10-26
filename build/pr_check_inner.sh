#!/bin/bash

set -exv

# copy the workspace from the Jenkins job off the ro volume into this container
mkdir /container_workspace
cp -r /workspace/. /container_workspace
cd /container_workspace

export KUBEBUILDER_ASSETS=/container_workspace/testbin/bin

(
  set -x; cd "$(mktemp -d)" &&
  OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
  KREW="krew-${OS}_${ARCH}" &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
  tar zxvf "${KREW}.tar.gz" &&
  ./"${KREW}" install krew
)

source build/template_check.sh

export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"

chmod 600 minikube-ssh-ident

ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube delete"
ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube start --cpus 6 --disk-size 10GB --memory 16000MB"

export MINIKUBE_IP=`ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube ip"`

scp -o StrictHostKeyChecking=no -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.key ./
scp -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.crt ./
scp -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/ca.crt ./

ssh -o ExitOnForwardFailure=yes -f -N -L 127.0.0.1:8444:$MINIKUBE_IP:8443 -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST

cat > kube-config <<- EOM
apiVersion: v1
clusters:
- cluster:
    certificate-authority: $PWD/ca.crt
    server: https://127.0.0.1:8444
  name: 127-0-0-1:8444
contexts:
- context:
    cluster: 127-0-0-1:8444
    user: remote-minikube
  name: remote-minikube
users:
- name: remote-minikube
  user:
    client-certificate: $PWD/client.crt
    client-key: $PWD/client.key
current-context: remote-minikube
kind: Config
preferences: {}
EOM

export PATH="$KUBEBUILDER_ASSETS:$PATH"
export PATH="/root/go/bin:$PATH"

export KUBECONFIG=$PWD/kube-config
kubectl config use-context remote-minikube
kubectl get pods --all-namespaces=true

source build/kube_setup.sh

export IMAGE_TAG=`git rev-parse --short HEAD`

kubectl create namespace clowder-system
kubectl apply -f clowder-config.yaml -n clowder-system

mkdir artifacts

cat manifest.yaml > artifacts/manifest.yaml

sed -i "s/clowder:latest/clowder:$IMAGE_TAG/g" manifest.yaml

kubectl apply -f manifest.yaml --validate=false

# Wait for operator deployment...
kubectl rollout status deployment clowder-controller-manager -n clowder-system

kubectl krew install kuttl

set +e
source build/run_kuttl.sh --report xml
KUTTL_RESULT=$?
mv kuttl-test.xml artifacts/junit-kuttl.xml

CLOWDER_PODS=$(kubectl get pod -n clowder-system -o jsonpath='{.items[*].metadata.name}')
for pod in $CLOWDER_PODS; do
    kubectl logs $pod -n clowder-system > artifacts/$pod.log
    kubectl logs $pod -n clowder-system | ./parse-controller-logs > artifacts/$pod-parsed-controller-logs.log
done

STRIMZI_PODS=$(kubectl get pod -n strimzi -o jsonpath='{.items[*].metadata.name}')
for pod in $STRIMZI_PODS; do
    kubectl logs $pod -n strimzi > artifacts/$pod.log
done
set -e

exit $KUTTL_RESULT
