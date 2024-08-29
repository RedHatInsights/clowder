# Web Provider

The ***Web Provider*** is responsible for creating the Service for the
public/private port and adding the port to the container template on the
deployment.

## ClowdApp Configuration

The public and private ports can be enabled by using the `webServices` stanza
on the `ClowdApp` specification.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  # Other App Config
  deployments:
    name: inventory
    podSpec: 
      image: quay.io/psav/clowder-hello
    webServices:
      public:
        enabled: true
        apiPath: hello
        whitelistPaths:
        - /api/hello/openapi.json
      private:
        enabled: true
```

## ClowdEnv Configuration

The **Web Provider** will run in one of the following modes. These are set up by
the ClowdEnvironment. Depending on the environment you are running you may or
may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

### operator

In operator mode, the **Web Provider** will set the port and service for a
deployment.

ClowdEnv Config options available:

- `port`
- `privatePort`
- `apiPrefix`
- `BOPURL`

### local

In local mode, the **Web Provider** will setup an entire mocked backend including
SSO, BOP and an aggregated gateway.

All pods which have the `webServices.public.enabled` set to `true` will also
have an auth pod injected into them which will be configured to work with the
SSO server and BOP URL. This will have a new port added to the service which
the gateway will be configured to use.

The `apiPath` parameter sets the URL that the service will be routed for. `/api/<apiPath>` will
be configured to route to the `auth` service port for that deployment.

The `whitelistPaths` parameter sets the paths that will not be required to go through authentication. These paths will always be able to be hit without auth. The following declarations are possible

- /absolute/path
- *prefixed/path
- /suffixed/path*
- *

#### mTLS cert-auth
Clowder also creates a cert-auth based gateway which can handle the mTLS flow
that is used in ConsoleDot for client machines. This creates a new gateway pod
which uses the same image as the sidecar, and configures an individual route
for each web service.

NOTE:
Just because a service is accessible via cert-auth in *local* mode, does not mean it is 
accessible in stage/production. Right now, there is no distinction between apps
that are and apps that are not available. This could change in the future, and would
require a ClowdApp flag addition.

The OpenShift or Minikube cluster is configured to passthrough SSL to this 
gateway, which enforcs strict SNI and requires mTLS.

##### Registration of cert
To facilitate cert-auth to be used in an environment the client must first
register the `CN` presented inside the clients cert, which has been signed by the
CA recognised by the Caddy Gateway. By default this CA is the `candlepin-ca` so
that machines may register to the real RHSM and use this cert in a local
instance.

NOTE: 
A *local* instance here corresponds to a ClowdEnvironment that has the *Web
Provider* set to *local* mode. This is the case for ephemeral environments.

The registration must be performed using an existing user account in the org,
and that user must be an `orgAdmin`. The default user provided with the *local* 
setup, has these permissions.

The registration command should look similar to the following, notice the UUID
is the same in all places for this example and **MUST** follow a UUID format:

```
> cat /tmp/test.json
{"uid": "36f23107-9b7c-48f6-8d5b-e6691e7dd235", "display_name": "36f23107-9b7c-48f6-8d5b-e6691e7dd235"}

> curl -k http://environment-host/v1/registrations -H "Authorization: Basic $EPHEM_BASE64" -vvv -H "Content-Type: application/json" -d @/tmp/test.json -H "x-rh-certauth-cn:/CN=36f23107-9b7c-48f6-8d5b-e6691e7dd235"
```

The `orgID` of the user credentials used to call the registrations endpoint will
be used to register this `CN`.

NOTE:
In systems registered with RHSM, there is an `orgId` present in the certificate,
but this will be ignored when registering with the registrations endpoint, as it
is not relevant to a local environment, which has been provisioned with it's own
Keycloak and hence its own `orgIDs`.

##### Making API calls with certs
The client cert/key combination can now be used to make API requests to services
via a new hostname with `-cert` appended. An example of this is shown below

```
> curl https://environment-host-cert/api/puptoo/ -vvv --key /tmp/tls.key --cert /tmp/tls.crt"
```

ClowdEnv Config options available:

- `port`
- `privatePort`
- `apiPrefix`
- `authPort`
- `gatewayCert`

## Generated App Configuration

The Metrics configuration appears in the cdappconfig.json with the following
structure.

### JSON structure

```json
{
  "publicPort": 8000,
  "privatePort": 10000,
  "apiPrefix": "/api"
}
```

### Client access

For supported languages, the web configuration is access via the following
attribute names.

| Language   | Attribute Name             |
|------------|----------------------------|    
| Python     | `LoadedConfig.publicPort`  |
| Go         | `LoadedConfig.PublicPort`  |
| Javascript | `LoadedConfig.publicPort`  |
| Ruby       | `LoadedConfig.publicPort`  |

### ClowdEnv Configuration

The **Web Provider** can be configured to set the public port, private port and
path as follows in this example.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
    web:
      mode: operator
      privatePort: 10000
      port: 8000
```

#### TLS Auth
The **Web Provider** also features a TLS sidecar option which will dynamically create and append an
Caddy sidecar to the deployment pod. This requires enabling in the configuration with an example
below.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
    web:
      # As above
      tls:
        enabled: true
        port: 18000
        privatePort: 18800
```

This configuration will do several things:

* Creates TLS annotations on the deployment's `Service` resource to allow *OpenShift* to create the
cert `Secret`
* Add new ports to the `Service` resource based on if the app has **public** or **private** ports 
enabled
* Creates a `ConfigMap` for the *Caddy* sidecar
* Adds the *Caddy* sidecar to the app's pod
** Sets VolumeMounts/Volumes for the config
** Sets VolumeMounts/Volumes for the cert/key
* Adds VolumeMounts/Volumes to the app's deployment to mount the CA chain, which is available in
every namespace via *OpenShift*
* Adds container ports to the app's deployment based on if the app has **public** or **private** 
ports enabled

Applications are then able to connect to other services in the cluster using the enpoints listed
in the `cdappconfig.json`. A `tlsCAPath` field is in the `cdappconfig.json` to tell people where the 
CA cert chain can be found for connecting to other services. All certs are registered against the
full hostname including *namespace* and *svc*. These hostnames are present in full in the endpoints
list and should be taken from there.

#### Customizing Cert Auth
The ClowdEnvironment can be configured to work with both `acme` and `self-signed`
certs by using the `spec.provides.web.gatewayCert.certMode` flag.

Custom CA certs may also be used by supplying the correct configuration as detailed below.
The `localCaConfigMap` field should point to a ConfigMap in the env namespace and
is expected to have the CA in PEM format `ca.pem` field.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: myenv
spec:
  # Other Env Config
  providers:
    web:
      # As above
      gatewayCert:
        enabled: true
        certMode: self-signed
        localCAConfigMap: my-configmap
```

For `acme` cert generation an `emailAddress` field needs to be supplied with the email address
to use for the cert generation.
