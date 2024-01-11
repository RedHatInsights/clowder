# Deployment Provider

The **Deployment Provider** is responsible for creating the base Deployment
resource for a `ClowdApp`.

NOTE: If a deployment uses a PVC type volume, the rollout strategy 
      will automatically be switched to `Recreate`.

## ClowdApp Configuration

To request a Kafka topic, a ``ClowdApp`` would use the `database` stanza, a
partial example of which is shown below.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  deployments:
  - name: service
    podSpec:
      name: quay.io/psav/clowder-hello
```

## ClowdEnv Configuration

There is no configuration for this provider.
