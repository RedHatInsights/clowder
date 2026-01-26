# Sidecars Provider

The **Sidecars Provider** is responsible for adding containers to pods to imbue them with the
requested sidecar functionality. Currently Clowder only support *splunk* and *token-refresher*, which were requested by the RHSM
team.


## ClowdApp Configuration

To request a Sidecar to be added to a container, use the ``sidecars`` stanza. An example below enables the splunk sidecar.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  deployments:
  - name: test
    sidecars:
    - name: splunk
      enabled: true  
```

## ClowdEnv Configuration

In order to allow sidecars to operate, they must be enabled in the 
``ClowdEnvironment``. The example below shows how to enabled the splunk sidecar.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  providers:
    sidecars:
      splunk:
        enabled: True
```

## Sidecar configuration

### Token Refresher
The token refreser sidecar requires a secret to be created with the following variables:

* ``CLIENT_ID``
* ``CLIENT_SECRET``
* ``ISSUER_URL``
* ``URL``

### Otel Collector
The Otel Collector sidecar requires a configmap to be present in the namespace of the app,
called `<app-name>-otel-config`. This will bind to a volume on the side and should contain the
following configuration to allow the health checks to work:

```yaml
extensions:
  health_check:
  health_check/1:
    endpoint: "localhost:13133"
    path: "/"
```
