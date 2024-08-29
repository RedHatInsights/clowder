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

### Splunk
The Splunk sidecar requires two resources to be created and added to the k8s cluster.

* ``Secret/<appName>-splunk`` - This secret must contain the field called ``splunk.pem`` and contain the cert to connect to splunk.
* ``ConfigMap/<appName>-splunk`` - This ConfigMap contains the ``inputs.conf`` file using the format similar to that shown below

```
[default]
_meta = namespace::rhsm-ci
host = $decideOnStartup

[monitor:///var/log/app/access.log]
index = rh_rhsm
sourcetype = springboot_access
ignoreOlderThan = 5d
recursive = false
disabled = false

[monitor:///var/log/app/server.log]
index = rh_rhsm
sourcetype = springboot_server
ignoreOlderThan = 5d
recursive = false
disabled = false
```

The namespace will be filled in automatically from an environment variable

### Token Refresher
The token refreser sidecar requires a secret to be created with the following variables:

* ``CLIENT_ID``
* ``CLIENT_SECRET``
* ``ISSUER_URL``
* ``URL``
