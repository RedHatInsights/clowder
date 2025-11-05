# Operating Clowder

## Table of Contents

1. [What is Clowder?](#what-is-clowder)
2. [Architecture Overview](#architecture-overview)
3. [Operating Apps Managed by Clowder](#operating-apps-managed-by-clowder)

---

## What is Clowder?

Clowder is a Kubernetes operator designed to simplify application configuration management and infrastructure dependency provisioning for cloud-native applications. Originally developed for Red Hat Insights, Clowder abstracts away the complexity of managing application configurations across different environments and deployment scenarios.

### Key Benefits

**Simplified Configuration Management**: Clowder eliminates the need for applications to manage environment-specific configurations by providing a standardized configuration format (`cdappconfig.json`) that contains all necessary connection details, credentials, and service endpoints.

**Infrastructure Abstraction**: Applications no longer need to know whether they're connecting to a local development database, a managed cloud service, or an operator-managed instance. Clowder handles the complexity of different infrastructure providers through its modular provider system.

**Environment Consistency**: The same application code can run unchanged across development, staging, and production environments. Clowder ensures that applications receive the appropriate configuration for their target environment without code changes.

**Dependency Management**: Clowder automatically manages service dependencies, ensuring that applications have access to required infrastructure services (databases, message queues, object storage) and other application services they depend on.

### How Clowder Works

Clowder utilizes a common configuration format that is presented to each application, no matter the environment it is running in, enabling a far easier development experience. It governs many different aspects of an application's configuration from defining the port it should listen to for its main web service, to metrics, kafka and others. When using Clowder, the burden of identifying and defining dependency and core service credentials and connection information is removed.

The operator watches for changes to `ClowdApp` and `ClowdEnvironment` custom resources and automatically provisions the necessary infrastructure and generates appropriate configurations for each application.

### Operating Clowder

There are two primary aspects of operating Clowder: Operating the apps managed by Clowder and operating Clowder itself.

## Architecture Overview

### High-Level Architecture

Clowder is a Kubernetes operator that manages application configuration and infrastructure dependencies for cloud-native applications. The operator follows the standard Kubernetes controller pattern, continuously reconciling desired state with actual state.

#### Core Components

1. **Clowder Controller Manager**
   - Main operator process that watches for CRD changes
   - Reconciles ClowdApp and ClowdEnvironment resources
   - Manages application lifecycle and configuration generation
   - Runs as a deployment in the `clowder-system` namespace

2. **Custom Resource Definitions (CRDs)**
   - `ClowdEnvironment`: Cluster-scoped resource defining infrastructure providers
   - `ClowdApp`: Namespace-scoped resource defining application specifications
   - `ClowdJobInvocation`: Resource for managing job executions

3. **Provider System**
   - Modular architecture supporting different infrastructure modes
   - Providers: Database, Kafka, Object Storage, Logging, Metrics, Web, etc.
   - Each provider supports multiple modes (local, operator, app-interface)

4. **Webhooks**
   - Validation webhooks for ClowdApp resource validation
   - Mutation webhooks for pod injection and configuration

#### Data Flow

```
ClowdApp → Controller → Provider Logic → K8s Resources → Application Config
    ↓           ↓              ↓              ↓              ↓
 Spec      Reconcile     Infrastructure   Deployments   cdappconfig.json
```

#### Deployment Architecture

Clowder is deployed as a standard Kubernetes operator with the following components:

- **Controller Manager Deployment**: Main operator workload
- **Custom Resource Definitions**: API definitions for Clowder resources
- **RBAC**: Service accounts, cluster roles, and role bindings for operator permissions
- **Webhooks**: Validation and mutation webhooks with TLS certificates
- **ConfigMaps**: Operator configuration and feature flags
- **Services**: Webhook services and metrics endpoints

#### Configuration Management

Clowder generates a standardized configuration format (`cdappconfig.json`) that contains:
- Database connection information
- Kafka broker details and topic configurations
- Object storage bucket credentials
- Service endpoints for dependencies
- Logging and metrics configuration
- Feature flags and environment-specific settings

This configuration is mounted as a secret in each application pod at `/cdappconfig.json`.

## Operating Apps Managed by Clowder

### ClowdEnvironment

**abbreviated to [env] in k8s**

The ``ClowdEnvironment`` CRD is responsible for configuring key infrastruture services that the Clowder enabled
apps will interact with. It is a *cluster scoped* CRD and thus must have a unique name inside the
k8s cluster. For production environments it is usual to have only one ``ClowdEnvironment``, whereas
in other scenarios -- such as ephemeral testing -- Clowder enables the management of
multiple environments that operate completely independently from each other.

#### Providers

An environment's specification is broken into **providers**, which govern the creation of services, e.g. Kafka topics,
object storage, etc, that applications may depend on. The ``ClowdEnvironment`` CRD configures these
providers principally by making use of a provider's **mode**.

#### Modes

Providers often operate in different modes. As an example the Kafka provider can operate in three
different modes. In *local* mode, the Kafka provider deploys a single node Kafka/Zookeeper instance
inside the cluster and configures it to auto-create the topics. In *operator* mode, the provider
assumes a Strimzi Kafka instance is present and will create ``KafkaTopic`` CRs to provide the
topics.  In *app-interface* mode, no resources are deployed and it is assumed app-interface has
already created the requested topics. For more information on the configuration of each of these
providers and their modes, please see the relevant pages.

#### Target Namespace

Environmental resources, such as the Kafka/Zookeeper from the example in the *Modes* section, will
be placed in the ``ClowdEnvironment``'s target namespace. This is configured by setting the
``targetNamespace`` attribute of the ``ClowdEnvironment``. If it is omitted, a random target
namespace is generated instead. The name of this resource can be found by inspecting the
``status.targetNamespace`` of the ClowdEnvironment resource.

### ClowdApp

**abbreviated to [app] in k8s**

The ``ClowdApp`` CRD is responsible for configuring an application and is namespace scoped. Any
resources Clowder creates on behalf of the application will reside in the same namespace that the
``ClowdApp`` resources is applied to. As such the ``ClowdApp`` name must be unique within a
particular namespace.  An ``ClowdApp`` does not have to be placed in the ``ClowdEnvironment``'s
target namespace.

A ``ClowdApp`` may define multiple services inside it. These services, though defined by a
specification that is very similar to the k8s pod specification, will be deployed as individual
deployment resources.  Functionally, defining multiple applications in the same ``ClowdApp``
specification allows the sharing of some infrastructure dependencies such as databases.
Applications in different ClowdApp's should not expect to be able to share databases.

A ``ClowdApp`` is coupled to a ``ClowdEnvironment`` by the use of the ``envName`` parameter of the
``ClowdApp``. When Clowder configures applications, it will point them to the resources that are
defined in the coupled ``ClowdEnvironment``. As an example, if a ``ClowdApp`` requires the use of a
Kafka topic, the application will be configured to use the kafka broker that has been configured in
the coupled ClowdEnvironment, which could be a local, strimzi or app-interface managed Kafka
instance.

#### Dependencies

An application will usually require several dependencies in the form of either infrastructure
services e.g. Kafka, or other application services such as RBAC. 

Services such as RBAC will be other Clowder-managed applications and, as such, have an associated
``ClowdApp`` coupled to the ``ClowdEnvironment``. These are defined in the ``dependencies`` field of
the ``ClowdApp`` and take the form of the dependency's ``ClowdApp`` name. This will result in all of
the dependent services being listed in the application's configuration. If a dependent service
defines multiple pod specs with a web service exposed in its ``ClowdApp``, each of these will be
exposed to the requesting app.  A ``ClowdApp`` will not be deployed if any of its service
dependencies do not exist within the coupled ``ClowdEnvironment``.

Infrastructure dependencies, such as Kafka topics and object bucket storage, are defined in the
``ClowdApp`` spec. More information on each of them is defined in the [API specification](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-clowdappspec).

#### Created Resources

For each ``ClowdApp`` service, Clowder will create an ``apps.Deployment`` and a ``Service``
resource.  If the service has the ``web`` field set to true, the ``Service`` resource will
include a port definition for ``webPort`` as well as the standard ``metricsPort``. The actual values
of these are defined in the ``ClowdEnvironment`` by configuring the web and metric providers,
respectively. By default these are set to 8000 for the web service port and 9000 for the metrics
port.

Clowder will also set certain fields in the pod spec, inline with best practice, such as pull
policy, and anti-affinity.

Clowder creates a ``Secret`` resource that is named the same as the ``ClowdApp`` which will contain the generated configuration
for that app. This secret will be mounted at ``/cdappconfig.json`` and will be consumed by the app
to configure itself on startup.

Secrets may also be created for application dependencies such as databases and in-memory db
services.






The deployment process involves building and pushing the Clowder application image to Quay. The image uses a tag based off the commit hash at the tip of master. The app image is built using ``build_deploy.sh``.

#### Promoting clowder to prod

As stated above, promoting Clowder to production is done the same as any other app in app-interface,
but there are additional considerations given how Clowder code changes could cause widespread
rollouts across the target cluster. For example, if a field is added to every app's
``cdappconfig.json``, this will trigger every deployment to rollout a new version at virtually the
same time.  While this *shouldn't* cause a problem, promoters should be aware that such churn is
going to happen before promoting.

Another more disruptive example would be if the format of the name of services was changed.  Not
only would this trigger a rollout of all deployments, but old pods would no longer function properly
because the old hostname in their configuration is no longer valid.  A change like this should
either be done in a backwards-compatible way or be done in a planned outage window.

Despite those two examples, most changes to Clowder should not be very disruptive; just make sure
that extra care is taken to review all changes before promoting to production.
