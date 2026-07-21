# Dependencies Provider

The **Dependencies Provider** is responsible for passing the list of dependent
enpoints through into the app configuration.

## ClowdApp Configuration

There are two kinds of dependency, optional and mandatory. With mandatory
dependencies, the application will not complete reconciliation unless the
dependency exists. With an optional dependency, the application will complete
reconciliation and will have its configuration updated should the dependency
come online. This is performed via a config hash annotation update on the
deployment template.

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: myapp
spec:
  # Other App Config
  dependencies:
  - app_name1
  optionalDependencies:
  - app_name2
```

## ClowdEnv Configuration

There are no configuration options for this provider.

## Generated App Configuration

The Endpoint appear in the cdappconfig.json with the following structure.

A client helper is available for the endpoints and privateEndpoints.

### JSON structure

```yaml
{
  "endpoints": [
    {
      "name": "deployment1",
      "app": "app_name1",
      "hostname": "deployment1.svc",
      "port": 8000
    },
    {
      "name": "deployment2",
      "app": "app_name2",
      "hostname": "deployment2.svc",
      "port": 8000
    },
  ],
  "privateEndpoints": [
    {
      "name": "deployment1",
      "app": "app_name1",
      "hostname": "deployment1.svc",
      "port": 10000
    },
  ]
}
```

### Client access

### Client helpers

The following helpers present a nested dictionary type structure allowing the
client to look up an endpoint via the app/deployment name. As an example:

```python
LoadedConfig.dependencyEndpoints["app_name1"]["deployment1"]
```

For supported languages, the dependency configuration is access via the
following attribute names.

| Language   | Attribute Name                      |
|------------|-------------------------------------|
| Python     | `LoadedConfig.dependencyEndpoints`  |
| Go         | `LoadedConfig.DependencyEndpoints`  |
| JavaScript | `LoadedConfig.dependencyEndpoints`  |
| Ruby       | `LoadedConfig.dependencyEndpoints`  |

Private endpoints are accessible via these attribute names.

| Language   | Attribute Name                             |
|------------|--------------------------------------------|
| Python     | `LoadedConfig.privateDependencyEndpoints`  |
| Go         | `LoadedConfig.PrivateDependencyEndpoints`  |
| JavaScript | `LoadedConfig.privateDependencyEndpoints`  |
| Ruby       | `LoadedConfig.privateDependencyEndpoints`  |

## V2 Dependency Endpoints

V2 dependency endpoints provide a simplified, URI-based format alongside the V1 flat-list
format. They appear in `cdappconfig.json` under `dependencyEndpoints.v2` (public) and
`privateDependencyEndpoints.v2` (private), nested by app name and deployment name.

### V2 JSON structure

```json
{
  "dependencyEndpoints": {
    "v2": {
      "app_name1": {
        "deployment1": {
          "uri": "http://app_name1-deployment1.namespace.svc:8000",
          "authenticated": false
        }
      },
      "app_name2": {
        "deployment1": {
          "uri": "https://app_name2-deployment1.namespace.svc:8443",
          "authenticated": false,
          "ca_certificate": "/cdapp/certs/service-ca.crt"
        }
      }
    }
  },
  "privateDependencyEndpoints": {
    "v2": {
      "app_name1": {
        "deployment1": {
          "uri": "http://app_name1-deployment1.namespace.svc:10000",
          "authenticated": false
        }
      }
    }
  }
}
```

### V2 endpoint fields

| Field            | Type   | Required | Description |
|------------------|--------|----------|-------------|
| `uri`            | string | yes      | Complete endpoint URI including protocol, hostname, and port. Uses `http://` for plaintext or `https://` for TLS. |
| `authenticated`  | bool   | yes      | Whether the client should authenticate when connecting. See [Authenticated field](#authenticated-field) below. |
| `ca_certificate` | string | no       | Path to CA certificate file for TLS verification. Only present for in-cluster (`ClowdApp`) TLS endpoints. Omitted for `ClowdAppRef` endpoints (which use the system trust store) and for plaintext endpoints. |

### Authenticated field

The `authenticated` field signals to clients whether they should present credentials (e.g.,
a service account token) when connecting to this dependency endpoint.

**Defaults:**

| Dependency type | Default `authenticated` | Rationale |
|-----------------|-------------------------|-----------|
| `ClowdApp` (in-cluster) | `false` | In-cluster communication relies on network isolation. |
| `ClowdAppRef` (cross-cluster) | `true` | Cross-cluster traffic typically routes through gateways requiring authentication. |

**Per-deployment override:**

The default can be overridden on the dependency's deployment via
`webServices.public.authenticated` (public endpoints) and
`webServices.private.authenticated` (private endpoints). This follows the same `*bool`
pattern as the `tls` field:

- Omitted (`nil`) — use the default for the dependency type.
- `true` — mark as requiring authentication (opt-in for `ClowdApp`).
- `false` — mark as not requiring authentication (opt-out for `ClowdAppRef`).

**Example — ClowdApp opt-in (in-cluster service requiring authentication):**

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: rbac
spec:
  deployments:
    - name: service
      webServices:
        public:
          enabled: true
          authenticated: true
        private:
          enabled: true
          authenticated: true
```

**Example — ClowdAppRef opt-out (ephemeral/mTLS environment):**

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdAppRef
metadata:
  name: rbac
spec:
  deployments:
    - name: service
      hostname: rbac.remote.example.com
      webServices:
        public:
          enabled: true
          authenticated: false
```
