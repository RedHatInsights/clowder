#!/usr/bin/env bash

# Get cpu and memory specs
cpu_count=$(cat /proc/cpuinfo | grep 'processor.*: [[:digit:]]$' | wc -l)
memory=$(($(cat /proc/meminfo | grep MemTotal | cut -f 2 -d : | tr -d " kB") / 1024))

# Download and install minikube
if [ -x /usr/local/bin/minikube ]; then
    echo "Minikube already installed"
    # TODO: Check for upgrades
else
    echo Downloading minikube...
    curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
    echo
    echo Installing minikube.  You will be asked for sudo password.
    sudo install minikube-linux-amd64 /usr/local/bin/minikube
fi

# Install bonfire
pip3 show crc-bonfire 2>/dev/null >/dev/null

if [ $? != "0" ]; then
    echo Installing Bonfire...
    echo
    pip3 install --user --upgrade crc-bonfire
fi

# Start minikube based on specs
minikube status 2>/dev/null >/dev/null
if [ $? != "0" ]; then
    echo Starting Minikube...
    echo
    cmd="minikube start --cpus=$(($cpu_count / 2)) --memory $(($memory / 2))MB --addons=registry --addons=ingress --driver=kvm2"
    echo $cmd
    $cmd
fi

echo Installing Clowder into Minikube...
echo
# TODO: Create a "latest" release and use it instead
minikube kubectl -- apply --validate=false -f https://github.com/RedHatInsights/clowder/releases/download/0.12.0/clowder-manifest-0.12.0.yaml
echo Installing ClowdEnvironment...
# TODO: Configure way to install pull secret
# TODO: NodePort config
bonfire process-env --clowd-env test 2>/dev/null | minikube kubectl -- apply --validate=false -f -
echo Creating test namespace...
minikube kubectl create ns test
echo Installing apps...
# TODO: Set up app dependencies such that only one app needs to be listed
bonfire process --clowd-env=test host-inventory ingress engine 2>/dev/null | minikube kubectl -- apply --validate=false -n test -f -
# TODO: Use 'minikube service' to get a URL(s) to the apps
# TODO: Install UI + keycloak
