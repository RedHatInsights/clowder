# ConfigHash Provider

The **ConfigHash Provider** is responsible for creating the configuration hash
annotation on the `Deployment` resource. Changes to configuration then modify
the deployment resource's template annotations and thereby restart pods,
forcing them to pick up the new configuration.

There is no configuration for this provider.

## Secret and ConfigMap restart triggers

Clowder has a handler that watches all secrets and configmaps. If a secret or
configmap has the `qontract.recycle` annotation set to `true` then the 
secret's contents are hashed and kept in a cache. When reconciling an app, 
the **ConfigHash Provider** will iterate over all *environment variables* and 
*volumes* to see if any ConfigMaps or Secrets are used for the app. If they 
are, then the **ConfigHash Provider** adds the ClowdApp to the list of dependant
apps for that secret.

Future updates to the secret will then trigger reconciliations for each 
ClowdApp listed. Each secret is hashed, and then the list of these hashes is 
also hashed to provude a unique fingerprint that is injected into the 
`cdappconfig.json`. If this value is different than it was before, the 
`configHash` annotation on the pod will be updated and this will restarted the
pod.
