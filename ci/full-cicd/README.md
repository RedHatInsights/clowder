Build container:
```
podman build -t clowder-cicd:latest .
```

Invoke tests (you must log into the OpenShift cluster first):
```
podman run -v ~/.kube/config:/opt/app-root/src/.kube/config:z -ti clowder-cicd:latest ./run.sh
```
