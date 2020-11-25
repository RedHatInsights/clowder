Operating Clowder
=================

Two primary aspects of operating Clowder: Operating the apps managed by Clowder
and operating Clowder itself.

Contrary to many other deployment models, where apps configure themselves, an application that is
managed by Clowder, will have configuration presented to it and is expected to use it. Clowder
governs many different aspects of an applications configuration from defining the port it should
listen to for its main web service, to metrics, kafka and others. In this way, a common configuration
format is presented to the application no matter the envrionment it is running in, enabling a far
easier development experience.

Operating Apps Managed by Clowder
---------------------------------

ClowdEnvironment
++++++++++++++++

**abbreviated to `env` in k8s**

The `ClowdEnvironment` CRD is responsible for configuring the environment that the Clowder enabled
apps will interact with. It is a *cluster scoped* CRD and thus must have a unique name inside the k8s
cluster. For production environments it is usual to have only one `ClowdEnvironment`, whereas in
other scenarios, such as ephemeral testing Clowder enables the management of multiple environments
that operate completely independently of each other.

Providers
^^^^^^^^^
Clowder provides a number of providers which govern the creation of services, e.g. Kafka Topics,
Object Storage, etc, that applications may depend on. The `ClowdEnvironment` CRD configures these
providers principally by making use of a provider's mode.

Modes
^^^^^
Providers often operate in different modes. As an example the Kafka Provider can operate in three
different modes. In *local* mode, the Kafka Provider deploys a single node kafka/zookeeper instance
inside the cluster and configures it to auto create the topics. In *operator* mode, the provider
assumes a Strimzi Kafka instance is present and will create KafkaTopic CRs to provide the topics.
In *app-interface* mode, no resources are deployed and it is assumed app-interface has already
created the requested topics. For more information on the configuration of each of these providers 
and their modes, please see the links below.

- `Kafka <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-databaseconfig>`_
- `Logging <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-loggingconfig>`_
- `Object storage <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-objectstoreconfig>`_
- `In-memory DB <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-inmemorydbconfig>`_
- `Relational DB <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-databaseconfig>`_
- `Web <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-webconfig>`_
- `Metrics <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-metricsconfig>`_

Target Namespace
^^^^^^^^^^^^^^^^
Clowder will provision two types of resources during operation, *environmental* and *application*
specific. Application resources will be deployed into the same namespace the ClowdApp resides in.
Environmental resources, such as the kafka/zookeeper from the exmaple in the *Modes* section, will
be placed in the ClowdEnvironment's target namespace. This is configured by setting the
`targetNamespace` attribute of the ClowdEnvironment and should always be set for production
environments. If it is omitted, a random target namespace is generated instead. The name of this
resource can be found by inspecting the `status.targetNamespace` of the ClowdEnvironment resource.

ClowdApp
++++++++

**abbreviated to `app` in k8s**

The `ClowdApp` CRD is responsible for configuring an application and is namespace scoped. Any resources
Clowder creates on behalf of the application will reside in the same namespace that the `ClowdApp`
resources is applied to. As such the `ClowdApp` name must be unique within a particular namespace.
An `ClowdApp` does not have to be placed in the `ClowdEnvironment`'s TargetNamespace.

A `ClowdApp` may define multiple services inside it. These services, though defined by a specification
that is very similar to the k8s Pod specification, will be deployed as individual Deployment resources.
Functionally, defining multiple applications in the same `ClowdApp` specification, allows the sharing 
of some infrastructure dependencies such as databases. Applications in different ClowdApp's should not
expect to be able to share databases.

A `ClowdApp` is coupled to a `ClowdEnvironment` by the use of the `envName` parameter of the `ClowdApp`
When Clowder configures applications it will point them to the resources that are defined in the
coupled `ClowdEnvironment`. As an example, if a `ClowdApp` requires the use of a Kafka topic, the 
application will be configured to use the kafka broker that has been configured in the coupled
ClowdEnvironment, which could be a local, strimzi or app-interface managed Kafka instance.

Dependencies
^^^^^^^^^^^^

An application will usually require several dependencies in the form of either infrastructure services
e.g. Kafka, or other application services such e.g. RBAC. 

Services, such as RBAC, will be other Clowder managed applications and as such, have an
associated `ClowdApp` coupled to the `ClowdEnvironment`. These are defined in the `dependencies` field
of the `ClowdApp` and take the form of the dependency's `ClowdApp` name. This will result in all
of the dependent services being exposed to the application's configuration. If a dependent service
defines multiple services in its ClowdApp, each of these will be exposed to the requesting app.

Infrastructure dependencies, such as Kafka topics and object bucket storage, are defined in the `ClowdApp`
spec. More information on each of them is defined in the `API specification <https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdappspec>`_.

Created Resources
^^^^^^^^^^^^^^^^^

For each `ClowdApp` service, Clowder will create an `apps.Deployment` and a `core.Service` resource.
If the service has the `web` field set to true, the `core.Service` resource will include a port
definition for `webPort` as well as the standard `metricsPort`. The actual values of these are defined
in the `ClowdEnvironment` by configuring the Web and Metric providers respectively. By default these
are set to 8000 for the web service port and 9000 for the metrics port.

Clowder will also set certain fields in the pod spec, inline with best practice, such as Pull
Policy, and Anti-Affinity.

Clowder will also create a `core.Secret` resource which will contain the generated configuration for
that app. This secret will be mounted at `/cdappconfig.json` and will be consumed by the app to configure
itself on startup.

Operating Clowder Itself
------------------------

- OLM pipeline
- Metrics and alerts
- App-interface modes
