# Architecture

This document describes the internal design of the Clowder Kubernetes operator: its components,
data flows, design tradeoffs, and implementation details. It is intended for contributors modifying
or extending the operator. Installation, usage, and development workflow belong in other documents.

## Table of Contents

1. [Overview](#overview)
2. [Custom Resource Definitions](#custom-resource-definitions)
3. [Controller Architecture](#controller-architecture)
4. [Provider System](#provider-system)
5. [Resource Cache](#resource-cache)
6. [Watch and Filter System](#watch-and-filter-system)
7. [Application Configuration Generation](#application-configuration-generation)
8. [Dependency Endpoint Resolution](#dependency-endpoint-resolution)
9. [Web Provider System](#web-provider-system)
10. [Observability](#observability)
11. [Code Generation Pipeline](#code-generation-pipeline)

---

## Overview

Clowder is a single Kubernetes operator (Go, controller-runtime/Kubebuilder) that manages the
common infrastructure concerns shared by cloud.redhat.com applications. Rather than each application
team building their own operator, Clowder provides a single opinionated control plane that handles
databases, messaging, object storage, in-memory caches, web services, metrics, logging, and
inter-application dependencies.

The central design choice is that application teams declare *what* they need (via `ClowdApp` CRs)
rather than *how* to provision it. The operator resolves this against an environment's provider
configuration (`ClowdEnvironment`) and assembles a complete runtime configuration document
(`cdappconfig.json`) that is injected into application containers as a mounted Kubernetes Secret.

The operator is structured around three CRDs, three controllers, and a pluggable provider system
where each capability area is implemented as an independently registered, ordered provider.

**Language:** Go 1.25

**Module:** `github.com/RedHatInsights/clowder`

---

## Custom Resource Definitions

All CRD types live in [`apis/cloud.redhat.com/v1alpha1/`][crd-dir].

### ClowdEnvironment

Cluster-scoped. Represents a single deployment environment (e.g., stage, production, ephemeral).
Each environment selects a mode and configuration for every provider type. No `ClowdApp` can exist
without a referenced `ClowdEnvironment`.

**Key spec fields (`ClowdEnvironmentSpec`):**

| Field | Purpose |
|-------|---------|
| `targetNamespace` | Namespace where environment-level resources (Kafka, Minio, etc.) are created |
| `providers` | `ProvidersConfig` — one config block per provider type (db, kafka, web, etc.) |
| `resourceDefaults` | Default pod resource requests/limits applied to any `ClowdApp` that omits them |
| `serviceConfig.type` | `ClusterIP` or `NodePort` for generated Services |
| `disabled` | Pauses reconciliation when true |

**Provider mode fields within `providers`:** Each provider sub-spec carries a `mode` enum that
selects the implementation. See [Provider System](#provider-system) for the full mode list per
provider.

**Status** tracks `targetNamespace`, `ready`, managed/ready deployment counts, managed/ready topic
counts, the list of apps in the env (`apps[]`), and a `prometheus.serverAddress`.

### ClowdApp

Namespace-scoped. Represents a single application composed of one or more Deployment or Job/CronJob
specs plus declarations of the infrastructure it needs.

**Key spec fields (`ClowdAppSpec`):**

| Field | Purpose |
|-------|---------|
| `envName` | References the owning `ClowdEnvironment` (cluster-scoped lookup by name) |
| `deployments[]` | List of `Deployment` objects; each produces a k8s `Deployment` + optional Services |
| `jobs[]` | List of `Job` objects; each produces a k8s `Job` or `CronJob` depending on `schedule` |
| `kafkaTopics[]` | Topic declarations; processed by the Kafka provider |
| `database` | Single DB spec (`name`, `version`, `sharedDbAppName`, t-shirt sizes) |
| `objectStore[]` | List of bucket names |
| `inMemoryDb` / `sharedInMemoryDbAppName` | In-memory DB request and optional sharing |
| `featureFlags` | Opt-in to feature flag provider config |
| `dependencies[]` | Required `ClowdApp` names; their endpoints are included in `cdappconfig.json` |
| `optionalDependencies[]` | Same as `dependencies` but do not block readiness if absent |
| `cyndi` | Cyndi pipeline configuration for database syndication (Kafka operator mode only) |
| `disabled` | Pauses reconciliation when true |

Each `Deployment` within a `ClowdApp` supports autoscaling via either a simple HPA
(`autoScalerSimple`) or a full KEDA `ScaledObject` (`autoScaler`). The naming pattern for all
generated resources is `{app-name}-{deployment-name}`.

**Status** tracks `ready`, managed/ready deployment counts, and `conditions` (including
`ReconciliationSuccessful` and `DeploymentsReady`).

### ClowdJobInvocation

Namespace-scoped. Triggers ad-hoc execution of one or more `Job` specs defined in a `ClowdApp`.
Also used to launch IQE (integration quality engineering) test jobs.

**Key spec fields (`ClowdJobInvocationSpec`):**

| Field | Purpose |
|-------|---------|
| `appName` | The `ClowdApp` that owns the jobs to invoke |
| `jobs[]` | Names of jobs (from `ClowdApp.spec.jobs`) to run |
| `testing.iqe` | IQE test job spec (plugin, image tag, UI test config) |
| `runOnNotReady` | Run jobs even if the app's deployments are not ready |
| `disabled` | Prevents the CJI from running |

**Status** tracks `completed`, and a `jobMap` of job name → state (`Invoked`, `Complete`, `Failed`).

### Relationships

```
ClowdEnvironment (cluster-scoped)
  └── ClowdApp (namespace-scoped, spec.envName → env name)
        └── ClowdJobInvocation (namespace-scoped, spec.appName → app name)
```

`ClowdApp` and `ClowdJobInvocation` resources set owner references pointing to their respective
parents, enabling garbage collection of generated resources.

---

## Controller Architecture

The three controllers live in [`controllers/cloud.redhat.com/`][ctrl-dir]:

- [`clowdenvironment_controller.go`][env-ctrl] — watches `ClowdEnvironment`
- [`clowdapp_controller.go`][app-ctrl] — watches `ClowdApp`
- [`clowdjobinvocation_controller.go`][cji-ctrl] — watches `ClowdJobInvocation`

Each controller is registered with the manager in [`run.go`][run-go].

### Reconciliation lifecycle

Each controller delegates to a dedicated reconciliation struct that executes a sequential list of
steps. For `ClowdApp`, the steps in `ClowdAppReconciliation.steps()` are:

1. Fetch the `ClowdApp` resource.
2. Update the set of "present and managed" apps.
3. Start Prometheus reconciliation duration metrics.
4. Handle deletion finalizers.
5. Check if the app is locked, disabled, or its namespace is being deleted.
6. Fetch the associated `ClowdEnvironment`.
7. Verify the environment's namespace is not being deleted.
8. Verify the environment has been reconciled and is ready.
9. Create the in-memory resource cache.
10. Run all providers in registration order.
11. Apply the cache atomically to Kubernetes.
12. Update deployment status on the `ClowdApp`.
13. Delete unused (orphaned) resources.
14. Set `ReconciliationSuccessful` condition and emit a success event.
15. Stop metrics.

`ClowdEnvironment` reconciliation follows the same step pattern but runs environment-level provider
functions (`EnvProvide()`) rather than app-level ones. It also creates and manages the
`targetNamespace` if it does not exist.

### Dual-reconciliation design quirk

When a `ClowdApp` is reconciled, the controller first calls `EnvProvide()` on every registered
provider before calling `Provide(app)`. This is because environment-level providers populate data
into the resource cache and the `AppConfig` struct (e.g., Kafka broker addresses, object store
endpoints) that app-level providers then read. Without re-running `EnvProvide()` during app
reconciliation, an app provider would see stale or missing infrastructure data.

This means that environment provider logic runs on every app reconciliation. Providers that perform
expensive or mutating environment setup must be idempotent and cache-aware to avoid side effects
from this double execution.

---

## Provider System

### Plugin model

All providers live under [`controllers/cloud.redhat.com/providers/`][providers-dir]. Each provider
package implements the `ClowderProvider` interface and registers itself using an `init()` function.

### Interface

```go
type ClowderProvider interface {
    EnvProvide() error        // called during environment reconciliation and before Provide()
    Provide(app *crd.ClowdApp) error  // called for each ClowdApp reconciliation
    GetConfig() *config.AppConfig
}
```

### Registration and ordering

Providers register via:

```go
providers.ProvidersRegistration.Register(setupFn, order, name)
```

The registry sorts by `order` integer. During reconciliation the registry is iterated in sorted
order. Providers with lower numbers run first. The ordering is critical: providers that produce
resources or config data must run before providers that consume that data.

**Registered providers and their execution order:**

| Order | Package | Name | Purpose |
|-------|---------|------|---------|
| 0 | `deployment` | deployment | Base Deployment/PodSpec construction |
| 0 | `cronjob` | cronjob | CronJob/Job construction |
| 1 | `web` | web | Web services, Services, Caddy config |
| 2 | `metrics` | metrics | Metrics Services, ServiceMonitors, Prometheus |
| 4 | `dependencies` | dependencies | Inter-app endpoint resolution |
| 5 | `database` | db | Database provisioning or credential pass-through |
| 5 | `featureflags` | featureflags | Unleash/feature flag provisioning |
| 5 | `inmemorydb` | inmemorydb | Redis or Elasticache |
| 5 | `logging` | logging | CloudWatch credentials or null logging |
| 5 | `objectstore` | objectstore | Minio or S3 credentials |
| 6 | `kafka` | kafka | Strimzi, MSK, local, managed Kafka |
| 7 | `reverseproxy` | reverseproxy | Frontend reverse proxy (ephemeral mode) |
| 10 | `autoscaler` | autoscaler | KEDA ScaledObject or simple HPA |
| 50 | `namespace` | namespace | Namespace-level resources |
| 97 | `serviceaccount` | serviceaccount | ServiceAccount creation |
| 98 | `pullsecrets` | pullsecrets | Pull secret attachment |
| 98 | `servicemesh` | servicemesh | Service mesh annotations |
| 98 | `sidecar` | sidecar | OTel collector and token refresher sidecars |
| 99 | `confighash` | confighash | Config Secret + hash annotation on Deployments |

### Provider modes

Each provider type supports multiple modes configured on `ClowdEnvironment.spec.providers.*`:

| Provider | Modes |
|----------|-------|
| `db` | `local` (deploy PostgreSQL), `app-interface` (external secret lookup), `shared` (schema isolation on shared DB), `none` |
| `kafka` | `operator` (Strimzi), `local` (ephemeral), `app-interface` (pass-through), `managed` (MSK via secret), `ephem-msk` (ephemeral MSK), `none` |
| `objectstore` | `minio` (deploy Minio), `app-interface` (S3 credentials from secret), `none` |
| `inmemorydb` | `redis` (deploy Redis), `elasticache` (credential secret lookup), `none` |
| `featureflags` | `local` (deploy Unleash), `app-interface` (credential pass-through), `none` |
| `web` | `none`/`operator` (Services only), `local` (full Keycloak + BOP stack) |
| `metrics` | `operator` (ServiceMonitors + Prometheus), `app-interface`, `none` |
| `logging` | `app-interface` (CloudWatch credentials), `null`, `none` |
| `reverseproxy` | `ephemeral`, `none` |
| `autoscaler` | `enabled` (KEDA), `none` |

The provider `init()` function selects the correct implementation struct based on the mode value at
setup time.

---

## Resource Cache

Clowder uses an in-memory `ObjectCache` (from `github.com/RedHatInsights/rhc-osdk-utils/resourceCache`)
rather than issuing Kubernetes API calls directly during provider execution.

### Why the cache exists

Without the cache, each provider would independently `Create` or `Update` Kubernetes objects. This
causes redundant reconciliation triggers (each API write fires watch events) and risks race
conditions between providers operating on the same object. The cache batches all writes and applies
them in a single controlled pass after all providers complete.

### How it works

1. At the start of reconciliation, a fresh `ObjectCache` is created.
2. Providers call `cache.Create(ident, nn, obj)` to register a new object, or `cache.Update(ident, obj)` to update an existing one. `cache.Get` and `cache.List` retrieve objects from the cache without hitting the API.
3. Objects are keyed by a `ResourceIdent` (a typed identifier scoped to a provider and resource name) plus a `NamespacedName`.
4. After all providers have run, `cache.ApplyAll()` issues the actual Kubernetes `Create` or `Update` calls in a defined apply order: `*` (all other types), then `Service`, `Secret`, `Deployment`, `Job`, `CronJob`, `ScaledObject`.
5. `cache.Reconcile(uid, opts)` lists existing resources matching owner labels and deletes any that were not written into the cache during this reconciliation pass, removing orphaned resources.

### Resource identification

Each resource type is declared as a `ResourceIdent` at package level in its provider:

```go
var CoreConfigSecret = rc.NewSingleResourceIdent(ProvName, "config_secret", &core.Secret{})
```

The `StrictGVK` option on the cache configuration causes it to reject registration of resource
types that were not declared in `AddPossibleGVKFromIdent` at provider setup time, preventing
accidental writes.

---

## Watch and Filter System

### Handler structure

All three controllers use a shared custom event handler: `enqueueRequestForObjectCustom`
(defined in [`controllers/cloud.redhat.com/handlers.go`][handlers-go]). This handler is created per
controller via `createNewHandler` and implements the `handler.EventHandler` interface.

The handler is responsible for:

- Determining the owning `ClowdApp` or `ClowdEnvironment` of an event-generating object via owner
  references.
- Deciding whether to enqueue a reconciliation request based on filtering logic.
- Maintaining the hash cache for ConfigMaps and Secrets.

### Event types and filter behavior

| Event | Filter logic |
|-------|-------------|
| **Create** | If the object is a tracked ConfigMap/Secret, update its hash cache entry and enqueue all apps using it. If the object has an owner of the watched kind, call `CreateFunc` handler to decide if a reconcile is needed. |
| **Update** | If the object carries the restarter annotation, update hash and re-enqueue owning apps. If the object is already in the hash cache (tracked), call `doUpdateToHash`. For generation-based resources, `UpdateFunc` is gated on generation change. |
| **Delete** | Remove object from hash cache. Enqueue the owning `ClowdApp`/`ClowdEnvironment` via `DeleteFunc`. |
| **Generic** | Enqueue owning resource via `GenericFunc` if present. |

The `HandlerFuncs` struct (with `CreateFunc`, `UpdateFunc`, `DeleteFunc`, `GenericFunc`) is
constructed per controller and encodes controller-specific rules — for example, gating Deployment
updates on generation change to avoid reconciling on status subresource writes.

### Hash cache for ConfigMaps and Secrets

The `HashCache` ([`controllers/cloud.redhat.com/hashcache/`][hashcache-dir]) is a thread-safe
in-memory structure that maps Kubernetes object identities to computed content hashes. Two separate
instances are created — one for `ClowdApp` and one for `ClowdEnvironment` — in `run.go`.

**Purpose:** When a Secret or ConfigMap that a pod depends on changes content, the hash cache
detects the change and enqueues reconciliation for all `ClowdApp` or `ClowdEnvironment` objects
that reference that object. During reconciliation, the `confighash` provider annotates Deployment
pods with the aggregate hash of all their dependent Secrets/ConfigMaps, causing Kubernetes to
roll out new pods when configuration changes.

**Key operations:**

- `CreateOrUpdateObject(obj, alwaysUpdate)` — computes a SHA256 hash of the object's content and stores it. Returns true if the hash changed.
- `AddClowdObjectToObject(clowdObj, obj)` — records that a `ClowdApp` or `ClowdEnvironment` depends on the given object.
- `GetSuperHashForClowdObject(clowdObj)` — returns an aggregate hash of all objects associated with a given `ClowdApp` or `ClowdEnvironment`.
- `RemoveClowdObjectFromObjects(obj)` — called at the start of each reconciliation to clear stale associations.
- `Delete(obj)` — removes an object from the cache on delete events.

Objects that carry the restarter annotation (configurable via operator config) are always tracked,
regardless of whether a provider explicitly registered them.

---

## Application Configuration Generation

### Schema-to-types pipeline

The canonical configuration schema lives at [`schema/schema.json`][schema-json]. Go types are
generated from this schema into [`controllers/cloud.redhat.com/config/types.go`][config-types] via:

```
make genconfig
```

This runs `gojsonschema` against `controllers/cloud.redhat.com/config/schema.json` (the schema is
also copied there during the build process). The generated `AppConfig` struct and its sub-types are
the shared data model that all providers write into during reconciliation.

### cdappconfig.json assembly

Each provider populates fields on the `config.AppConfig` struct held in `Provider.Config`. Because
providers run in order, later providers can read data written by earlier ones. The `confighash`
provider (order 99, last to run) serializes the completed `AppConfig` to JSON and writes it as the
value of `cdappconfig.json` in a Kubernetes Secret:

```
Secret: {app-name}-cdappconfig (in the app namespace)
Key: cdappconfig.json
Value: <JSON-serialized AppConfig>
```

The `confighash` provider also computes the aggregate hash of all dependent Secrets and ConfigMaps
and annotates all managed Deployments with this hash, triggering rolling updates on config changes.

### Secret mounting

The generated `cdappconfig.json` Secret is mounted into every pod managed by the `ClowdApp`. The
mount path is determined by the `ACG_CONFIG` environment variable convention. Applications use a
language-specific client library (e.g., `app-common-go`) to parse this file at startup.

---

## Dependency Endpoint Resolution

### Declaration

A `ClowdApp` declares dependencies on other apps via `spec.dependencies` (required) and
`spec.optionalDependencies` (present when available). Both are lists of `ClowdApp` names within
the same `ClowdEnvironment`.

### Resolution mechanism

The `dependencies` provider (order 4) resolves endpoints as follows:

1. Lists all `ClowdApp` resources in the same environment via `env.GetAppsInEnv()`.
2. Also lists `ClowdAppRef` resources (lightweight references for apps not fully managed by Clowder).
3. For each declared dependency, finds the matching app and extracts its public and private web
   service endpoints.
4. Populates `config.AppConfig.Endpoints` (public) and `config.AppConfig.PrivateEndpoints` for
   the app being reconciled.

Resolution always includes the app's own endpoints first (self-registration), then external
dependencies.

### Endpoint types

Each resolved endpoint carries:

- `hostname` — the Kubernetes Service DNS name
- `port` — the public web port
- `tlsPort` — the TLS port, if TLS is enabled for that deployment
- `app` — the source app name
- `name` — the deployment name within the source app

Private endpoints (`PrivateDependencyEndpoint`) similarly carry `hostname`, `port`, `tlsPort`, and
`app`/`name` identifiers.

### TLS per endpoint

TLS configuration is per-endpoint, not global. Each `Deployment` within a `ClowdApp` can
individually enable or disable TLS for its public and private services via
`webServices.public.tls` and `webServices.private.tls`, overriding the environment-level TLS
default. Dependent apps receive the CA certificate path for each individual endpoint in
`cdappconfig.json`, enabling mutual TLS with granular trust boundaries.

---

## Web Provider System

### Provider types

The web provider is selected based on `ClowdEnvironment.spec.providers.web.mode`:

| Mode | Implementation | Description |
|------|----------------|-------------|
| `none` / `operator` | `webProvider` (`default.go`) | Creates Kubernetes Services for each enabled web service. No gateway deployed. |
| `local` | `localWebProvider` (`local.go`) | Deploys a full local auth stack: Keycloak, mock BOP, mock entitlements, and a Caddy gateway. |

### Caddy gateway

In `local` mode a Caddy-based reverse proxy gateway is deployed in the environment's
`targetNamespace`. It reads routing configuration from a generated ConfigMap and terminates TLS
at the gateway boundary. Gateway certificate modes are configurable:

- `self-signed` — Caddy generates a self-signed certificate.
- `acme` — Caddy uses Let's Encrypt ACME with a configured email address.
- `none` — no gateway TLS.

The gateway ConfigMap is assembled from all registered ClowdApp API paths and updated on every
relevant reconciliation.

### H2C support

HTTP/2 cleartext (H2C) is supported at the per-deployment level via `webServices.public.h2cEnabled`
and `webServices.private.h2cEnabled`. When enabled, the web provider creates an additional Service
port on the environment's `h2cPort` / `h2cPrivatePort`. The target port for H2C services can be
overridden per deployment via `h2cTargetPort`.

### Service naming

Generated Services follow a deterministic naming pattern:

- Public service: `{app-name}-{deployment-name}`
- Private service: `{app-name}-{deployment-name}-private`

This pattern is also used by the dependency resolution provider when constructing endpoint hostnames.

---

## Observability

### Prometheus metrics

The `metrics` provider manages Prometheus integration. In `operator` mode it creates:

- A `Prometheus` custom resource (via the Prometheus Operator) in the environment's target namespace.
- A `ServiceAccount`, `Role`, and `RoleBinding` for the Prometheus instance.
- A `ServiceMonitor` for each app that has metrics enabled.
- Optionally, a Prometheus Pushgateway deployment and associated `ServiceMonitor`.

In `app-interface` mode, `ServiceMonitor` resources are still created but the Prometheus instance
itself is not deployed (it is assumed to exist externally).

The Prometheus server address is written to `ClowdEnvironment.status.prometheus.serverAddress` for
consumption by other components.

### OpenTelemetry

The `sidecar` provider (order 98) can inject an OpenTelemetry collector sidecar into every pod
managed by a `ClowdApp`. This is configured environment-wide via
`ClowdEnvironment.spec.providers.sidecars.otelCollector.enabled`, with per-app override capability
via `ClowdApp.spec.deployments[].podSpec.sidecars[]`.

The OTel collector sidecar:

- Uses a configurable image (operator config → environment spec → app spec override, in priority
  order).
- Reads its pipeline configuration from a per-app ConfigMap (default name: `{app-name}-otel-config`),
  which can be overridden at environment or app level.
- Supports per-app environment variable injection and memory resource limits.

A token refresher sidecar (`tokenRefresher`) is also available under the same provider, used for
service account token renewal in environments requiring short-lived credentials.

### Operator-internal metrics

The operator itself exposes Prometheus metrics (registered in the controller setup code) tracking:

- Per-provider reconciliation duration (`providerMetrics`), labeled by provider name and source
  (`clowdapp` or `clowdenvironment`).
- Reconciliation counts and durations per controller.

---

## Code Generation Pipeline

Three separate generation steps produce different artifacts. They are independent and must be run in
the correct sequence when both API types and configuration schema change.

### `make generate`

**Runs:** `controller-gen object:headerFile=...`

**Produces:** `apis/cloud.redhat.com/v1alpha1/zz_generated.deepcopy.go`

**Why:** Kubebuilder/controller-runtime requires all types stored in the API server to implement
`runtime.Object`, which in practice means implementing `DeepCopyObject()`. The code generator reads
`+kubebuilder:object:root=true` markers on type definitions and generates the full DeepCopy method
tree. This must be re-run whenever a field is added, removed, or changed in any CRD type file.

### `make manifests`

**Runs:** `controller-gen crd rbac:roleName=... webhook`

**Produces:**
- `config/crd/bases/` — CRD YAML manifests (one per CRD)
- `config/rbac/` — ClusterRole YAML derived from `+kubebuilder:rbac` markers in controller files

**Why:** The CRD YAML must match the Go type definitions exactly. Kubebuilder markers embedded in
the type files (e.g., `+kubebuilder:validation:Enum`, `+kubebuilder:printcolumn`) control
validation, printer columns, and scope. The RBAC manifests ensure the operator's service account
has exactly the permissions its controllers declare via `+kubebuilder:rbac` comments.

### `make genconfig`

**Runs:** `gojsonschema -p config -o types.go schema.json`

**Source:** `controllers/cloud.redhat.com/config/schema.json` (canonical source: `schema/schema.json`)

**Produces:** `controllers/cloud.redhat.com/config/types.go`

**Why:** The `cdappconfig.json` document consumed by application containers has a versioned JSON
Schema. Generating Go types from the schema ensures the operator's internal `AppConfig` struct
stays in sync with the published schema without manual maintenance. The schema is the source of
truth; the Go types are derived artifacts. This must be re-run whenever the config schema changes.

### Ordering constraint

When making changes that span both CRD types and config schema:

1. Edit type files in `apis/cloud.redhat.com/v1alpha1/`.
2. Run `make generate` to regenerate DeepCopy.
3. Run `make manifests` to regenerate CRD YAML and RBAC.
4. Edit `schema/schema.json` if configuration fields changed.
5. Run `make genconfig` to regenerate `config/types.go`.

Running `make manifests` before `make generate` will produce inconsistent CRD manifests if new
types lack DeepCopy implementations.

---

[crd-dir]: apis/cloud.redhat.com/v1alpha1/
[ctrl-dir]: controllers/cloud.redhat.com/
[env-ctrl]: controllers/cloud.redhat.com/clowdenvironment_controller.go
[app-ctrl]: controllers/cloud.redhat.com/clowdapp_controller.go
[cji-ctrl]: controllers/cloud.redhat.com/clowdjobinvocation_controller.go
[run-go]: controllers/cloud.redhat.com/run.go
[providers-dir]: controllers/cloud.redhat.com/providers/
[handlers-go]: controllers/cloud.redhat.com/handlers.go
[hashcache-dir]: controllers/cloud.redhat.com/hashcache/
[schema-json]: schema/schema.json
[config-types]: controllers/cloud.redhat.com/config/types.go
