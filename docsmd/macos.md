# Using Clowder on MacOS

## Install Minikube

``brew install minikube``

If you do not use ``brew``, you can follow [this guide](https://v1-18.docs.kubernetes.io/docs/tasks/tools/install-minikube/)


## Install HyperKit or VirtualBox

Virtualbox will work, but we recommend HyperKit. It is much faster and more 
light weight than VirtualBox, but VirtualBox will also work just fine. 

``brew install hyperkit``

or 

Install VirtualBox from [the VirtualBox site](https://www.virtualbox.org/wiki/Downloads)


## Running

Minikube can now be run the same way as the rest of the documentation suggests. 
Setting the config will also make the minikube experience less verbose.
``minikube config set vm-driver hyperkit`` or  ``minikube config set vm-driver virtualbox``.
