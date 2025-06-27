# In-Memory DB Provider

The **In-Memory DB Provider** is responsible for providing access to an in-memory
DB instance.

## ClowdApp Configuration

To request an in-memory db instance, a `ClowdApp` would use the `inMemoryDb`
stanza, a partial example of which is shown below.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  # Other App Config
  inMemoryDb: true
```

## ClowdEnv Configuration

The **In-Memory DB Provider** will run in one of the following modes. These are set up by
the ClowdEnvironment. Depending on the environment you are running you may or
may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

### redis

In redis mode, the **In-Memory DB Provider** will provision a single node redis instance
in the same namespace as the ``ClowdApp`` that requested it.


### elasticache

In elasticache mode, the **In-Memory DB Provider** will search for a secret named
`in-memory-db` inside the same namespace as the `ClowdApp` that requested it.
The hostname and port will then be passed to the `cdappconfig.json` for use by
the app. If a password is provided, it is known that in-transit encryption is enabled, as per [ElastiCache requirements](https://docs.aws.amazon.com/AmazonElastiCache/latest/dg/auth.html#auth-using).

## shared

In shared mode, the **In-Memory DB Provider** will use the **redis** instance defined
in the `ClowdApp` referenced by the `SharedInMemoryDbAppName` configuration option.

## Generated App Configuration

The In-Memory DB configuration appears in the cdappconfig.json with the
following structure.

### JSON structure

```json
  "inMemoryDb": {
    "hostname": "hostname",
    "port": 27015,
    "username": "username",
    "password": "password"
  }
}
```

### Client access

For supported languages, the In-Memory DB configuration is accessed via the following
attribute names.

| Language   | Attribute Name             |
|------------|----------------------------|
| Python     | `LoadedConfig.inMemoryDb`  |
| Go         | `LoadedConfig.InMemoryDb`  |
| JavaScript | `LoadedConfig.inMemoryDb`  |
| Ruby       | `LoadedConfig.inMemoryDb`  |



### ClowdEnv Configuration

Configuring the **In-Memory DB Provider** is done by providing the follow JSON
structure to the ``ClowdEnv`` resource. Further details of the options
available can be found in the API reference. A minimal example is shown below
for the ``redis`` mode. Different modes can use different configuration
options, more information can be found in the API reference.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
    inMemoryDb:
      mode: redis
```
