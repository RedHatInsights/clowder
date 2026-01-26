# Database Provider

The **Database Provider** is responsible for providing access to a PostgreSQL
database.

## ClowdApp Configuration

To request a database, a `ClowdApp` would use the `database` stanza, a
partial example of which is shown below.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  # Other App Config
  database:
    name: inventory
    version: 12
```

### Using a Shared Database across multiple ClowdApps

To share a database from one ClowdApp to another Clowder supports sharing a database 
declared in one ClowdApp with many others.

Request a database like above, but then in your dependent ClowdApp resource set up
the shared configuration like so:

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp-worker
spec:
  # Other App Config
  database:
    sharedDbAppName: myapp
  dependencies:
  - myapp
```

This example would set up `myapp-worker` looking at the same database as `myapp`.
The strings need to be the same.

## ClowdEnv Configuration

### Modes

The **Database Provider** will run in one of the following modes. These are set up
by the ClowdEnvironment. Depending on the environment you are running you may
or may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

#### local

In local mode, the **Database Provider** will provision a single node PostgreSQL
instance for every app that requests a database and place it in the same
namespace as the `ClowdApp`. The client will be given credentials for both a
normal user and an admin user.

ClowdEnv Config options available:

- `pvc`

#### shared

In shared mode, the **Database Provider** will provision a single node PostgreSQL
and configure every app to use the same instance. As in the local mode, the client
will be given credentials for both a normal and an admin user.

ClowdEnv Config options available:
- `pvc`

#### app-interface

In app-interface mode, the Clowder operator does not create any resources and
simply passes through configuration from a secret to the client config. First
the provider will search for any secrets that have the annotation of
``clowder/database: <app-name>`` where the app-name matches the ClowdApp name.
If this cannot be found then the provider will search all secrets in the same
namespace looking for a hostname which is of the form
`<name>-<env>.*********` where `name` is the name defined in the
`ClowdApp` `database` stanza, and `env` is usually one of either
`stage` or `prod`.

## Generated App Configuration

The Database configuration appears in the cdappconfig.json with the following
structure. As well as the hostname and port, credentials and database name are
presented.

A client helper is available for the RDS CA, used in app-interface mode.

### JSON structure

```yaml
{
    "database": {
    "name": "dBaseName",
    "username": "username",
    "password": "password",
    "hostname": "hostname",
    "port": 5432,
    "pgPass": "testing",
    "adminUsername": "adminusername",
    "adminPassword": "adminpassword",
    "rdsCa": "ca"
    }
}
```

### Client access

For supported languages, the database configuration is access via the following
attribute names.

| Language    | Attribute                 |
|-------------|---------------------------|
| Python      | `LoadedConfig.database`   |
| Go          | `LoadedConfig.Database`   |
| JavaScript  | `LoadedConfig.database`   |
| Ruby        | `LoadedConfig.database`   |



### Client helpers

#### **RDS Ca**

Returns a filename which points to a temporary file containing the
contents of the CA cert.

| Language    | Attribute                  |
|-------------|----------------------------|
| Python      | `LoadedConfig.rds_ca()`    |
| Go          | `LoadedConfig.RdsCa()`     |
| JavaScript  | Not yet implemented        |
| Ruby        | Not yet implemented        |

### ClowdEnv Configuration

Configuring the **Database Provider** is done by providing the follow JSON
structure to the `ClowdEnv` resource. Further details of the options
available can be found in the API reference. A minimal example is shown below
for the `operator` mode. Different modes can use different configuration
options, more information can be found in the API reference.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
  database:
    mode: local
    pvc: false
```
