# Service Account Provider

The **Service Account Provider** is responsible for creating the service
accounts for each deployment to operate under. It creates one app service
account for all app infrastructure deploymenes, i.e. redis, database, one env
service account for any env infrastructure deployments and one service account
per deployment inside a ClowdApp. This is to allow individual deployments to
have more granular access to the k8s API.

## ClowdApp Configuration

Service accounts are created automatically, and require no configuration,
however, setting a `ClowdApp` to have the `k8sAccessLevel` as shown below, will
grant that deployment elevated k8sAccessLevel. The available options are
`default`, `view` and `edit`.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  # Other App Config
  deployments:
    podSpec:
      name: inventory
    k8sAccessLevel: "edit"
```

## ClowdEnv Configuration

There is no configuration for this provider.
