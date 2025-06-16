# Using Clowder on MacOS

## Install Minikube

``brew install minikube``

If you do not use ``brew``, you can follow [this guide](https://v1-18.docs.kubernetes.io/docs/tasks/tools/install-minikube/)


## Install Podman

Virtualbox or HyperKit were previously recommended, but Podman is becoming a popular option. Hyperkit has been deprecated due to lack of upstream maintenance. Podman support is "experimental" at this time, but works reliably enough for locally reproducing issues. Once you have Podman installed, you can establish it as the driver with something like this (adjust your parameters accordingly):

``minikube start --cpus 4 --disk-size 36GB --memory 16000MB --driver=podman --addons registry --addons ingress  --addons=metrics-server --disable-optimizations``

## Virtualbox or Hyperkit (deprecated)

``brew install hyperkit`` (you will see a warning about the project being deprecated)

or 

Install VirtualBox from [the VirtualBox site](https://www.virtualbox.org/wiki/Downloads)


## Running

Minikube can now be run the same way as the rest of the documentation suggests. 
Setting the config will also make the minikube experience less verbose.

``minikube config set vm-driver podman``
