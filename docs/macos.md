# Using Clowder on MacOS

## Install Minikube

``brew install minikube``

If you do not use ``brew``, you can follow [this guide](https://v1-18.docs.kubernetes.io/docs/tasks/tools/install-minikube/)


## Run minikube

```
kube start \
    --cpus 4 \
    --disk-size 36GB \
    --memory 16000MB \
    --driver=docker \
    --addons=registry \
    --addons=ingress \
    --addons=metrics-server \
    --disable-optimizations \
    --container-runtime=containerd \
    --kubernetes-version=v1.32.6 \
    --insecure-registry "10.0.0.0/24"
```
    
## Configure Minikube for Local Testing
```
build/setup_kube.sh
```

## Setup Network Proxy

Next start an Alpine docker container that will act as a proxy that will connect your local machine's port 5000 to the registry running in the Minikube cluster at port 5000. You'll find more on this in the Minikube documentation for running [docker on macOS](https://minikube.sigs.k8s.io/docs/handbook/registry/#enabling-insecure-registries).

```
docker run --rm -it -d --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"
```

## Run

Lastly, run the make target `deploy-minikube-quick` that will build the image locally, push the image to the registry running in Minikube, and start the pod. This command also sets a tag that will need to be updated each time you make a change locally to Clowder. This will ensure that the new pod comes up with your changes. 
```
CLOWDER_BUILD_TAG=boop334 make deploy-minikube-quick
```

