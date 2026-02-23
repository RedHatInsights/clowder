# Logging Provider

The **Logging Provider** is responsible for providing configuration for logging

## ClowdApp Configuration

Logging configuration is automatically passed through to a the client
configuration and so no request is made in the `ClowdApp`

## ClowdEnv Configuration

The **Logging Provider** will run in one of the following modes. These are set up by
the ClowdEnvironment. Depending on the environment you are running you may or
may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

### app-interface

In `app-interface` mode, the **Logging Provider** will search for a secret called
`cloudwatch` in the same namespace as the `ClowdApp` and present the
configuration into the `cdappconfig.json`

## Generated App Configuration

The Logging configuration appears in the cdappconfig.json with the following
structure. The example below is given for `app-interface` mode.

### JSON structure

```yaml
{
  "logging": {
    "type": "cloudwatch",
    "cloudwatch": {
      "accessKeyId": "ACCESS_KEY",
      "secretAccessKey": "SECRET_ACCESS_KEY",
      "region": "EU",
      "logGroup": "base_app"
    }
  }
}
```

### Client Access

For supported languages, the logging configuration is access via the following
attribute names.

| Language   | Attribute Name          |
|------------|-------------------------|
| Python     | `LoadedConfig.logging`  |
| Go         | `LoadedConfig.Logging`  |
| Javascript | `LoadedConfig.logging`  |
| Ruby       | `LoadedConfig.logging`  |

### ClowdEnv Configuration

The only configuration for the **Logging Provider** is the mode.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
    logging:
      mode: app-interface
```
