# Operating Clowder

Two primary aspects of operating Clowder: Operating the apps managed by Clowder and operating
Clowder itself.

Clowder utilizes a common configuration format that is presented to each application, no matter
the environment it is running in, enabling a far easier development experience. It governs many
different aspects of an applications configuration from defining the port it should listen to for
its main web service, to metrics, kafka and others. When using Clowder, the burden of identifying
and defining dependency and core service credentials and connection information is removed.

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

Environmental resources, such as the Kafka/Zookeeper from the exmaple in the *Modes* section, will
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
``ClowdApp`` spec. More information on each of them is defined in the [API specification](https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdappspec).

#### Created Resources

For each ``ClowdApp`` service, Clowder will create an ``apps.Deployment`` and a ``Service``
resource.  If the service has the ``web`` field set to true, the ``Service`` resource will
include a port definition for ``webPort`` as well as the standard ``metricsPort``. The actual values
of these are defined in the ``ClowdEnvironment`` by configuring the web and metric providers,
respectively. By default these are set to 8000 for the web service port and 9000 for the metrics
port.

Clowder will also set certain fields in the pod spec, inline with best practice, such as pull
policy, and anti-affinity.

Clowder creates a ``Secret`` resource which will contain the generated configuration
for that app. This secret will be mounted at ``/cdappconfig.json`` and will be consumed by the app
to configure itself on startup.

Secrets may also be created for application dependencies such as databases and in-memory db
services.

## Operating Clowder Itself

### OLM pipeline

Clowder is deployed via OLM, thus the build and deploy pipeline comprises of creating and deploying
OLM CRs: ``OperatorGroup``, ``CatalogSource``, and ``Subscription``.  To truly understand OLM is
outside the scope of this document, but it will cover how each resource is managed. 

Despite being deployed via OLM, Clowder follows a very similar build and deployment pipeline as
other apps in app-interface, specifically all pushes to master are automatically deployed to stage,
and an MR to app-interface is required to update the ref in production.

``OperatorGroup`` and ``Subscription`` are quite static, but ``CatalogSource`` is what gets updated
every promotion.  Before it's updated, there are three images that are pushed to Quay:  the Clowder
application image, the Clowder OLM bundle, and the Clowder catalog image.  All three images use the
same image tag, based off the commit hash at the tip of master.  The app image is built using
``build_deploy.sh``, and the bundle and catalog images are built in a separate Jenkins job using
``build_catalog.sh``.

#### Troubleshooting

On occasion, updating the ``CatalogSource`` does not trigger OLM to deploy the latest version of
Clowder.  If this happens, the simplest approach is to delete the ``ClusterServiceVersion`` and
``Subscription`` resources with the name ``clowder`` from the ``clowder`` namespace.  Once they are
removed, you should re-run the saas-deploy job for clowder, which will recreate the
``Subscription``, which should trigger OLM to recreate the ``ClusterServiceVersion``.

##### Metrics and alerts

##### App-interface modes

##### Promoting clowder to prod

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
