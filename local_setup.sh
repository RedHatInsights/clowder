#! /bin/bash

minikube start --cpus 8 --disk-size 36GB --memory 16000MB --driver=kvm2 --addons registry --addons ingress  --addons=metrics-server --disable-optimizations
# minikube start --cpus 4 --disk-size 36GB --memory 20000MB --driver=podman --addons registry --addons ingress  --addons=metrics-server --disable-optimizations

./build/kube_setup.sh

make install

make deploy-minikube

kubectl create ns jumpstart

bonfire deploy-env -n jumpstart
