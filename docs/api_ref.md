API Reference
=============

-   [cloud.redhat.com/v1alpha1](#k8s-api-cloud-redhat-com-v1alpha1)

-   [kafka.strimzi.io/v1beta1](#k8s-api-kafka-strimzi-io-v1beta1)

cloud.redhat.com/v1alpha1
-------------------------

Package v1alpha1 contains API Schema definitions for the
cloud.redhat.com v1alpha1 API group

-   [ClowdApp](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdapp)

-   [ClowdAppList](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdapplist)

-   [ClowdEnvironment](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironment)

-   [ClowdEnvironmentList](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironmentlist)

### ClowdApp

ClowdApp is the Schema for the clowdapps API

-   [ClowdAppList](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdapplist)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`cloud.redhat.com/v1alpha1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`ClowdApp`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta[$$ObjectMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
<tr class="even">
<td>*`spec`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdappspec[$$ClowdAppSpec$$]__</td>
<td>A ClowdApp specification.</td>
</tr>
</tbody>
</table>

### ClowdAppList

ClowdAppList contains a list of ClowdApp

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`cloud.redhat.com/v1alpha1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`ClowdAppList`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta[$$ListMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
<tr class="even">
<td>*`items`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdapp[$$ClowdApp$$]__</td>
<td>A list of ClowdApp Resources.</td>
</tr>
</tbody>
</table>

### ClowdAppSpec

ClowdAppSpec is the main specification for a single Clowder Application
it defines n pods along with dependencies that are shared between them.

-   [ClowdApp](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdapp)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`pods`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-podspec[$$PodSpec$$]__</td>
<td>A list of pod specs</td>
</tr>
<tr class="even">
<td>*`envName`* __string__</td>
<td>The name of the ClowdEnvironment resource that this ClowdApp will use as its base. This does not mean that the ClowdApp needs to be placed in the same directory as the targetNamespace of the ClowdEnvironment.</td>
</tr>
<tr class="odd">
<td>*`kafkaTopics`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopicspec[$$KafkaTopicSpec$$]__</td>
<td>A list of Kafka topics that will be created and made available to all the pods listed in the ClowdApp.</td>
</tr>
<tr class="even">
<td>*`database`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-databasespec[$$DatabaseSpec$$]__</td>
<td>The database specification defines a single database, the configuration of which will be made available to all the pods in the ClowdApp.</td>
</tr>
<tr class="odd">
<td>*`objectStore`* __string array__</td>
<td>A list of string names defining storage buckets. In certain modes, defined by the ClowdEnvironment, Clowder will create those buckets.</td>
</tr>
<tr class="even">
<td>*`inMemoryDb`* __boolean__</td>
<td>If inMemoryDb is set to true, Clowder will configure pass configuration of an In Memory Database to the pods in the ClowdApp. This single instance will be shared between all apps.</td>
</tr>
<tr class="odd">
<td>*`dependencies`* __string array__</td>
<td>A list of dependencies in the form of the name of the ClowdApps that are required to be present for this ClowdApp to function.</td>
</tr>
</tbody>
</table>

### ClowdEnvironment

ClowdEnvironment is the Schema for the clowdenvironments API

-   [ClowdEnvironmentList](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironmentlist)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`cloud.redhat.com/v1alpha1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`ClowdEnvironment`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta[$$ObjectMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
<tr class="even">
<td>*`spec`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironmentspec[$$ClowdEnvironmentSpec$$]__</td>
<td>A ClowdEnvironmentSpec object.</td>
</tr>
</tbody>
</table>

### ClowdEnvironmentList

ClowdEnvironmentList contains a list of ClowdEnvironment

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`cloud.redhat.com/v1alpha1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`ClowdEnvironmentList`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta[$$ListMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
<tr class="even">
<td>*`items`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironment[$$ClowdEnvironment$$]__</td>
<td>A list of ClowdEnvironment objects.</td>
</tr>
</tbody>
</table>

### ClowdEnvironmentSpec

ClowdEnvironmentSpec defines the desired state of ClowdEnvironment.

-   [ClowdEnvironment](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironment)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`targetNamespace`* __string__</td>
<td>TargetNamespace describes the namespace where any generated environmental resources should end up, this is particularly important in (*_local_*) mode.</td>
</tr>
<tr class="even">
<td>*`providers`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig[$$ProvidersConfig$$]__</td>
<td>A ProvidersConfig object, detailing the setup and configuration of all the providers used in this ClowdEnvironment.</td>
</tr>
<tr class="odd">
<td>*`resourceDefaults`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#resourcerequirements-v1-core[$$ResourceRequirements$$]__</td>
<td>Defines the default resource requirements in standard k8s format in the event that they omitted from a PodSpec inside a ClowdApp.</td>
</tr>
</tbody>
</table>

### DatabaseConfig

DatabaseConfig configures the Clowder provider controlling the creation
of Database instances.

-   [ProvidersConfig](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`mode`* __DatabaseMode__</td>
<td>The mode of operation of the Clowder Database Provider. Valid options are: (*_app-interface_*) where the provider will pass through database credentials found in the secret defined by the database name in the ClowdApp, and (*_local_*) where the provider will spin up a local instance of the database.</td>
</tr>
<tr class="even">
<td>*`image`* __string__</td>
<td>In (*_local_*) mode, the Image field is used to define the database image for local database instances.</td>
</tr>
<tr class="odd">
<td>*`pvc`* __boolean__</td>
<td>If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes.</td>
</tr>
</tbody>
</table>

### DatabaseSpec

DatabaseSpec is a struct defining a database to be exposed to a
ClowdApp.

-   [ClowdAppSpec](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdappspec)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`version`* __integer__</td>
<td>Defines the Version of the PostGreSQL database, defaults to 12.</td>
</tr>
<tr class="even">
<td>*`name`* __string__</td>
<td>Defines the Name of the datbase to be created. This will be used as the name of the logical database inside the database server in (*_local_*) mode and the name of the secret to be used for Database configuration in (*_app-interface_*) mode.</td>
</tr>
</tbody>
</table>

### InMemoryDBConfig

InMemoryDBConfig configures the Clowder provider controlling the
creation of InMemoryDB instances.

-   [ProvidersConfig](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`mode`* __InMemoryMode__</td>
<td>The mode of operation of the Clowder InMemory Provider. Valid options are: (*_redis_*) where a local Minio instance will be created. This provider currently has no mode for app-interface.</td>
</tr>
<tr class="even">
<td>*`pvc`* __boolean__</td>
<td>If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes.</td>
</tr>
</tbody>
</table>

### InitContainer

InitContainer is a struct defining a k8s init container. This will be
deployed along with the parent pod and is used to carry out one time
initialization procedures.

-   [PodSpec](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-podspec)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`command`* __string array__</td>
<td>A list of commands to run inside the parent Pod.</td>
</tr>
<tr class="even">
<td>*`args`* __string array__</td>
<td>A list of args to be passed to the init container.</td>
</tr>
<tr class="odd">
<td>*`inheritEnv`* __boolean__</td>
<td>If true, inheirts the environment variables from the parent pod. specification</td>
</tr>
<tr class="even">
<td>*`env`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#envvar-v1-core[$$EnvVar$$] array__</td>
<td>A list of environment variables used only by the initContainer.</td>
</tr>
</tbody>
</table>

### KafkaConfig

KafkaConfig configures the Clowder provider controlling the creation of
Kafka instances.

-   [ProvidersConfig](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`clusterName`* __string__</td>
<td>Defines the cluster name to be used by the Kafka Provider this will be used in some modes to locate the Kafka instance.</td>
</tr>
<tr class="even">
<td>*`namespace`* __string__</td>
<td>The Namespace the cluster is expected to reside in. This is only used in (*_app-interface_*) and (*_operator_*) modes.</td>
</tr>
<tr class="odd">
<td>*`mode`* __KafkaMode__</td>
<td>The mode of operation of the Clowder Kafka Provider. Valid options are: (*_operator_*) which expects a Strimzi Kafka instance and will configure KafkaTopic CRs and place them in the Namespace described in the configuration, (*_app-interface_*) which simple passes the topic names through to the App's cdappconfig.json and expects app-interface to have created the relevant topics, and (*_local_*) where a small instance of Kafka is created in the Env namespace and configured to auto-create topics.</td>
</tr>
<tr class="even">
<td>*`suffix`* __string__</td>
<td>(Unused)</td>
</tr>
<tr class="odd">
<td>*`pvc`* __boolean__</td>
<td>If using the (*_local_*) mode and PVC is set to true, this instructs the local Kafka instance to use a PVC instead of emptyDir for its volumes.</td>
</tr>
</tbody>
</table>

### LoggingConfig

LoggingConfig configures the Clowder provider controlling the creation
of Logging instances.

-   [ProvidersConfig](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`mode`* __LoggingMode__</td>
<td>The mode of operation of the Clowder Logging Provider. Valid options are: (*_app-interface_*) where the provider will pass through cloudwatch credentials to the app configuration, and (*_none_*) where no logging will be configured.</td>
</tr>
</tbody>
</table>

### MetricsConfig

MetricsConfig configures the Clowder provider controlling the creation
of metrics services and their probes.

-   [ProvidersConfig](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`port`* __integer__</td>
<td>The port that metrics services inside ClowdApp pods should be served on. If omitted, defaults to 9000.</td>
</tr>
<tr class="even">
<td>*`path`* __string__</td>
<td>A prefix path that pods will be instructed to use when setting up their metrics server.</td>
</tr>
<tr class="odd">
<td>*`mode`* __MetricsMode__</td>
<td>The mode of operation of the Metrics provider. The allowed modes are (*_none_*), which disables metrics service generation, or (*_operator_*) where services and probes are generated.</td>
</tr>
</tbody>
</table>

### MinioStatus

MinioStatus defines the status of a minio instance in local mode.

-   [ObjectStoreStatus](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-objectstorestatus)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`credentials`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#secretreference-v1-core[$$SecretReference$$]__</td>
<td>A reference to standard k8s secret.</td>
</tr>
<tr class="even">
<td>*`hostname`* __string__</td>
<td>The hostname of a Minio instance.</td>
</tr>
<tr class="odd">
<td>*`port`* __integer__</td>
<td>The port number the Minio instance is to be served on.</td>
</tr>
</tbody>
</table>

### ObjectStoreConfig

ObjectStoreConfig configures the Clowder provider controlling the
creation of ObjectStore instances.

-   [ProvidersConfig](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`mode`* __ObjectStoreMode__</td>
<td>The mode of operation of the Clowder ObjectStore Provider. Valid options are: (*_app-interface_*) where the provider will pass through Amazon S3 credentials to the app configuration, and (*_minio_*) where a local Minio instance will be created.</td>
</tr>
<tr class="even">
<td>*`suffix`* __string__</td>
<td>Currently unused.</td>
</tr>
<tr class="odd">
<td>*`pvc`* __boolean__</td>
<td>If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes.</td>
</tr>
</tbody>
</table>

### ObjectStoreStatus

ObjectStoreStatus defines the status of a Minio setup in local mode,
including buckets.

-   [ClowdEnvironmentStatus](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironmentstatus)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`minio`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-miniostatus[$$MinioStatus$$]__</td>
<td>A MinioStatus object.</td>
</tr>
<tr class="even">
<td>*`buckets`* __string array__</td>
<td>A list of buckets provided by the Minio instance.</td>
</tr>
</tbody>
</table>

### PodSpec

PodSpec defines a container running inside a ClowdApp.

-   [ClowdAppSpec](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdappspec)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`name`* __string__</td>
<td>Name defines the identifier of a Pod inside the ClowdApp. This name will be used along side the name of the ClowdApp itself to form a &lt;app&gt;-&lt;pod&gt; pattern which will be used for all other created resources and also for some labels. It must be unique within a ClowdApp.</td>
</tr>
<tr class="even">
<td>*`minReplicas`* __integer__</td>
<td>Defines the minimum replica count for the pod.</td>
</tr>
<tr class="odd">
<td>*`image`* __string__</td>
<td>Image refers to the container image used to create the pod.</td>
</tr>
<tr class="even">
<td>*`initContainers`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-initcontainer[$$InitContainer$$]__</td>
<td>A list of init containers used to perform at-startup operations.</td>
</tr>
<tr class="odd">
<td>*`command`* __string array__</td>
<td>The command that will be invoked inside the pod at startup.</td>
</tr>
<tr class="even">
<td>*`args`* __string array__</td>
<td>A list of args to be passed to the pod container.</td>
</tr>
<tr class="odd">
<td>*`env`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#envvar-v1-core[$$EnvVar$$] array__</td>
<td>A list of environment variables in k8s defined format.</td>
</tr>
<tr class="even">
<td>*`resources`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#resourcerequirements-v1-core[$$ResourceRequirements$$]__</td>
<td>A pass-through of a resource requirements in k8s ResourceRequirements format. If omitted, the default resource requirements from the ClowdEnvironment will be used.</td>
</tr>
<tr class="odd">
<td>*`livenessProbe`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#probe-v1-core[$$Probe$$]__</td>
<td>A pass-through of a Liveness Probe specification in standard k8s format. If omited, a standard probe will be setup point to the webPort defined in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to false.</td>
</tr>
<tr class="even">
<td>*`readinessProbe`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#probe-v1-core[$$Probe$$]__</td>
<td>A pass-through of a Readiness Probe specification in standard k8s format. If omited, a standard probe will be setup point to the webPort defined in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to false.</td>
</tr>
<tr class="odd">
<td>*`volumes`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#volume-v1-core[$$Volume$$] array__</td>
<td>A pass-through of a list of Volumes in standa k8s format.</td>
</tr>
<tr class="even">
<td>*`volumeMounts`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#volumemount-v1-core[$$VolumeMount$$] array__</td>
<td>A pass-through of a list of VolumesMounts in standa k8s format.</td>
</tr>
<tr class="odd">
<td>*`web`* __boolean__</td>
<td>If set to true, creates a service on the webPort defined in the ClowdEnvironment resource, along with the relevant liveness and readiness probes.</td>
</tr>
</tbody>
</table>

### ProvidersConfig

ProvidersConfig defines a group of providers configuration for a
ClowdEnvironment.

-   [ClowdEnvironmentSpec](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdenvironmentspec)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`db`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-databaseconfig[$$DatabaseConfig$$]__</td>
<td>Defines the Configuration for the Clowder Database Provider.</td>
</tr>
<tr class="even">
<td>*`inMemoryDb`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-inmemorydbconfig[$$InMemoryDBConfig$$]__</td>
<td>Defines the Configuration for the Clowder InMemoryDB Provider.</td>
</tr>
<tr class="odd">
<td>*`kafka`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-kafkaconfig[$$KafkaConfig$$]__</td>
<td>Defines the Configuration for the Clowder Kafka Provider.</td>
</tr>
<tr class="even">
<td>*`logging`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-loggingconfig[$$LoggingConfig$$]__</td>
<td>Defines the Configuration for the Clowder Logging Provider.</td>
</tr>
<tr class="odd">
<td>*`metrics`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-metricsconfig[$$MetricsConfig$$]__</td>
<td>Defines the Configuration for the Clowder Metrics Provider.</td>
</tr>
<tr class="even">
<td>*`objectStore`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-objectstoreconfig[$$ObjectStoreConfig$$]__</td>
<td>Defines the Configuration for the Clowder ObjectStore Provider.</td>
</tr>
<tr class="odd">
<td>*`web`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-webconfig[$$WebConfig$$]__</td>
<td>Defines the Configuration for the Clowder Web Provider.</td>
</tr>
</tbody>
</table>

### WebConfig

WebConfig configures the Clowder provider controlling the creation of
web services and their probes.

-   [ProvidersConfig](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-providersconfig)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`port`* __integer__</td>
<td>The port that web services inside ClowdApp pods should be served on. If omitted, defaults to 8000.</td>
</tr>
<tr class="even">
<td>*`apiPrefix`* __string__</td>
<td>An api prefix path that pods will be instructed to use when setting up their web server.</td>
</tr>
<tr class="odd">
<td>*`mode`* __WebMode__</td>
<td>The mode of operation of the Web provider. The allowed modes are (*_none_*), which disables web service generation, or (*_operator_*) where services and probes are generated.</td>
</tr>
</tbody>
</table>

kafka.strimzi.io/v1beta1
------------------------

Package v1beta1 contains API Schema definitions for the kafka.strimzi.io
v1beta1 API group

-   [Kafka](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafka)

-   [KafkaList](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkalist)

-   [KafkaTopic](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopic)

-   [KafkaTopicList](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopiclist)

### Address

Address struct represents the physical connection details of a Kafka
Server.

-   [KafkaListener](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkalistener)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`host`* __string__</td>
<td>Host defines the hostname of the Kafka server.</td>
</tr>
<tr class="even">
<td>*`port`* __integer__</td>
<td>Port defines the port of the Kafka server.</td>
</tr>
</tbody>
</table>

### Condition

-   [KafkaStatus](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkastatus)

-   [KafkaTopicStatus](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopicstatus)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`lastTransitionTime`* __string__</td>
<td>The last transition time of the resource.</td>
</tr>
<tr class="even">
<td>*`message`* __string__</td>
<td>The message of the last transition.</td>
</tr>
<tr class="odd">
<td>*`reason`* __string__</td>
<td>The Reason for hte transition change.</td>
</tr>
<tr class="even">
<td>*`type`* __string__</td>
<td>The type of the condition.</td>
</tr>
</tbody>
</table>

### Kafka

Kafka is the Schema for the kafkatopics API

-   [KafkaList](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkalist)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`kafka.strimzi.io/v1beta1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`Kafka`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta[$$ObjectMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
</tbody>
</table>

### KafkaList

KafkaList contains a list of instances.

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`kafka.strimzi.io/v1beta1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`KafkaList`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta[$$ListMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
<tr class="even">
<td>*`items`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafka[$$Kafka$$]__</td>
<td>A list of Kafka objects.</td>
</tr>
</tbody>
</table>

### KafkaListener

KafkaListener represents a configured Kafka listener instance.

-   [KafkaStatus](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkastatus)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`addresses`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-address[$$Address$$]__</td>
<td>A list of addresses that the Kafka instance is listening on.</td>
</tr>
<tr class="even">
<td>*`bootstrapServers`* __string__</td>
<td>A bootstrap server that the Kafka client can initially conenct to.</td>
</tr>
<tr class="odd">
<td>*`type`* __string__</td>
<td>The Kafka server type.</td>
</tr>
<tr class="even">
<td>*`certificates`* __string array__</td>
<td>A list of certificates to be used with this Kafka instance.</td>
</tr>
</tbody>
</table>

### KafkaTopic

KafkaTopic is the Schema for the kafkatopics API

-   [KafkaTopicList](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopiclist)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`kafka.strimzi.io/v1beta1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`KafkaTopic`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta[$$ObjectMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
<tr class="even">
<td>*`spec`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopicspec[$$KafkaTopicSpec$$]__</td>
<td>The KafkaTopicSpec specification defines a KafkaTopic.</td>
</tr>
</tbody>
</table>

### KafkaTopicList

KafkaTopicList contains a list of KafkaTopic

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`apiVersion`* __string__</td>
<td>`kafka.strimzi.io/v1beta1`</td>
</tr>
<tr class="even">
<td>*`kind`* __string__</td>
<td>`KafkaTopicList`</td>
</tr>
<tr class="odd">
<td>*`metadata`* __link:https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta[$$ListMeta$$]__</td>
<td>Refer to Kubernetes API documentation for fields of `metadata`.</td>
</tr>
<tr class="even">
<td>*`items`* __xref:{anchor_prefix}-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopic[$$KafkaTopic$$]__</td>
<td>A list of KafkaTopic objects.</td>
</tr>
</tbody>
</table>

### KafkaTopicSpec

KafkaTopicSpec defines the desired state of KafkaTopic

-   [ClowdAppSpec](#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-clowdappspec)

-   [KafkaTopic](#k8s-api-cloud-redhat-com-clowder-v2-apis-kafka-strimzi-io-v1beta1-kafkatopic)

<table>
<colgroup>
<col style="width: 25%" />
<col style="width: 75%" />
</colgroup>
<thead>
<tr class="header">
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>*`config`* __object (keys:string, values:string)__</td>
<td>A key/value pair describing the configuration of a particular topic.</td>
</tr>
<tr class="even">
<td>*`partitions`* __integer__</td>
<td>The requested number of partitions for this topic.</td>
</tr>
<tr class="odd">
<td>*`replicas`* __integer__</td>
<td>The requested number of replicas for this topic.</td>
</tr>
<tr class="even">
<td>*`topicName`* __string__</td>
<td>The topic name.</td>
</tr>
</tbody>
</table>
