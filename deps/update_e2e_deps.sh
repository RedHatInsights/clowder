#!/bin/bash

cd deps/kustomize

echo """ 
-------------------------------------
updating kustomize dependency
---------------------------------------
"""
go get sigs.k8s.io/kustomize/kustomize/v4

cd ../controller-gen

echo """
--------------------------------------
updating controller-gen dependency
--------------------------------------
"""
go get sigs.k8s.io/controller-tools

cd ../..
