# Feature Flags Provider

The **Feature Flags Provider** is responsible for providing access to a feature
flags server.

## ClowdApp Configuration

To request a Feature Flags server, a ``ClowdApp`` would use the `featureFlags`
stanza, an example of which is shown below.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  featureFlags: true
```

## Feature Flags Modes

### local

In local mode, the **Feature Flags Provider** will provision an Unleash server. This
instance will be created when the ``ClowdEnv`` is deployed.

### app-interface

In app-interface mode, the **Feature Flags Provider** will look up the secret defined in the
environment spec and return the hostname, port and access token in the cdapp configuration.

## Generated App Configuration

The Feature Flags configuration appears in the cdappconfig.json with the
following structure. 

### JSON structure

```json
{
  "featureFlags": {
    "hostname": "ff-server.server.example.com",
    "port": 4242
    "clientAccessToken": "someaccesstoken"
  }
}
```

### Client access

For supported languages, the feature flags configuration is access via the
following attribute names.

Language  | Attribute Name               
--|--
Python    | ``LoadedConfig.featureFlags``
Go        | ``LoadedConfig.FeatureFlags``
Javscript | ``LoadedConfig.featureFlags``
Ruby      | ``LoadedConfig.featureFlags``


### ClowdEnv Configuration

Configuring the **Feature Flags Provider** is done by providing the follow JSON
structure to the ``ClowdEnv`` resource. Further details of the options
available can be found in the API reference. A minimal example is shown below
for the ``local`` mode. Different modes can use different configuration
options, more information can be found in the API reference.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
    featureFlags:
      mode: local
      pvc: false
```

App-interface mode requires a little more configuration:

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
    featureFlags:
      mode: app-interface
      hostname: app-interface-ff-server
      port: 1234
      credentialRef: the name of the app-interface Unleash credential secret
```
