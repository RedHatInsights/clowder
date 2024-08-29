# API Reference

## Packages
- [cloud.redhat.com/v1alpha1](#cloudredhatcomv1alpha1)


## cloud.redhat.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the cloud.redhat.com v1alpha1 API group

### Resource Types
- [ClowdApp](#clowdapp)
- [ClowdAppList](#clowdapplist)
- [ClowdEnvironment](#clowdenvironment)
- [ClowdEnvironmentList](#clowdenvironmentlist)
- [ClowdJobInvocation](#clowdjobinvocation)
- [ClowdJobInvocationList](#clowdjobinvocationlist)



#### APIPath

_Underlying type:_ _string_

A string representing an API path that should route to this app for Clowder-managed Ingresses (in format "/api/somepath/")

_Appears in:_
- [PublicWebService](#publicwebservice)



#### AppInfo



AppInfo details information about a specific app.

_Appears in:_
- [ClowdEnvironmentStatus](#clowdenvironmentstatus)

| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `deployments` _[DeploymentInfo](#deploymentinfo) array_ |  |


#### AppProtocol

_Underlying type:_ _string_

AppProtocol is used to define an appProtocol for Istio

_Appears in:_
- [PrivateWebService](#privatewebservice)



#### AppResourceStatus





_Appears in:_
- [ClowdAppStatus](#clowdappstatus)

| Field | Description |
| --- | --- |
| `managedDeployments` _integer_ |  |
| `readyDeployments` _integer_ |  |


#### AutoScaler



AutoScaler defines the autoscaling parameters of a KEDA ScaledObject targeting the given deployment.

_Appears in:_
- [Deployment](#deployment)

| Field | Description |
| --- | --- |
| `pollingInterval` _integer_ | PollingInterval is the interval (in seconds) to check each trigger on. Default is 30 seconds. |
| `cooldownPeriod` _integer_ | CooldownPeriod is the interval (in seconds) to wait after the last trigger reported active before scaling the deployment down. Default is 5 minutes (300 seconds). |
| `maxReplicaCount` _integer_ | MaxReplicaCount is the maximum number of replicas the scaler will scale the deployment to. Default is 10. |
| `minReplicaCount` _integer_ | MinReplicaCount is the minimum number of replicas the scaler will scale the deployment to. |
| `advanced` _[AdvancedConfig](#advancedconfig)_ |  |
| `triggers` _ScaleTriggers array_ |  |
| `fallback` _[Fallback](#fallback)_ |  |
| `externalHPA` _boolean_ | ExternalHPA allows replicas on deployments to be controlled by another resource, but will not be allowed to fall under the minReplicas as set in the ClowdApp. |


#### AutoScalerConfig



AutoScalerConfig configures the Clowder provider controlling the creation of AutoScaler configuration.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[AutoScalerMode](#autoscalermode)_ | Enable the autoscaler feature |


#### AutoScalerMode

_Underlying type:_ _string_

AutoScaler mode enabled or disabled the autoscaler. The key "keda" is deprecated but preserved for backwards compatibility

_Appears in:_
- [AutoScalerConfig](#autoscalerconfig)



#### AutoScalerSimple



SimpleAutoScaler defines a simple HPA with scaling for RAM and CPU by value and utilization thresholds, along with replica count limits

_Appears in:_
- [Deployment](#deployment)

| Field | Description |
| --- | --- |
| `replicas` _[SimpleAutoScalerReplicas](#simpleautoscalerreplicas)_ |  |
| `ram` _[SimpleAutoScalerMetric](#simpleautoscalermetric)_ |  |
| `cpu` _[SimpleAutoScalerMetric](#simpleautoscalermetric)_ |  |


#### ClowdApp



ClowdApp is the Schema for the clowdapps API

_Appears in:_
- [ClowdAppList](#clowdapplist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloud.redhat.com/v1alpha1`
| `kind` _string_ | `ClowdApp`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ClowdAppSpec](#clowdappspec)_ | A ClowdApp specification. |


#### ClowdAppList



ClowdAppList contains a list of ClowdApp



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloud.redhat.com/v1alpha1`
| `kind` _string_ | `ClowdAppList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[ClowdApp](#clowdapp) array_ | A list of ClowdApp Resources. |


#### ClowdAppSpec



ClowdAppSpec is the main specification for a single Clowder Application it defines n pods along with dependencies that are shared between them.

_Appears in:_
- [ClowdApp](#clowdapp)

| Field | Description |
| --- | --- |
| `deployments` _[Deployment](#deployment) array_ | A list of deployments |
| `jobs` _[Job](#job) array_ | A list of jobs |
| `envName` _string_ | The name of the ClowdEnvironment resource that this ClowdApp will use as its base. This does not mean that the ClowdApp needs to be placed in the same directory as the targetNamespace of the ClowdEnvironment. |
| `kafkaTopics` _[KafkaTopicSpec](#kafkatopicspec) array_ | A list of Kafka topics that will be created and made available to all the pods listed in the ClowdApp. |
| `database` _[DatabaseSpec](#databasespec)_ | The database specification defines a single database, the configuration of which will be made available to all the pods in the ClowdApp. |
| `objectStore` _string array_ | A list of string names defining storage buckets. In certain modes, defined by the ClowdEnvironment, Clowder will create those buckets. |
| `inMemoryDb` _boolean_ | If inMemoryDb is set to true, Clowder will pass configuration of an In Memory Database to the pods in the ClowdApp. This single instance will be shared between all apps. |
| `featureFlags` _boolean_ | If featureFlags is set to true, Clowder will pass configuration of a FeatureFlags instance to the pods in the ClowdApp. This single instance will be shared between all apps. |
| `dependencies` _string array_ | A list of dependencies in the form of the name of the ClowdApps that are required to be present for this ClowdApp to function. |
| `optionalDependencies` _string array_ | A list of optional dependencies in the form of the name of the ClowdApps that will be added to the configuration when present. |
| `testing` _[TestingSpec](#testingspec)_ | Iqe plugin and other specifics |
| `cyndi` _[CyndiSpec](#cyndispec)_ | Configures 'cyndi' database syndication for this app. When the app's ClowdEnvironment has the kafka provider set to (*_operator_*) mode, Clowder will configure a CyndiPipeline for this app in the environment's kafka-connect namespace. When the kafka provider is in (*_app-interface_*) mode, Clowder will check to ensure that a CyndiPipeline resource exists for the application in the environment's kafka-connect namespace. For all other kafka provider modes, this configuration option has no effect. |
| `disabled` _boolean_ | Disabled turns off reconciliation for this ClowdApp |




#### ClowdEnvironment



ClowdEnvironment is the Schema for the clowdenvironments API

_Appears in:_
- [ClowdEnvironmentList](#clowdenvironmentlist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloud.redhat.com/v1alpha1`
| `kind` _string_ | `ClowdEnvironment`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ClowdEnvironmentSpec](#clowdenvironmentspec)_ | A ClowdEnvironmentSpec object. |


#### ClowdEnvironmentList



ClowdEnvironmentList contains a list of ClowdEnvironment



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloud.redhat.com/v1alpha1`
| `kind` _string_ | `ClowdEnvironmentList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[ClowdEnvironment](#clowdenvironment) array_ | A list of ClowdEnvironment objects. |


#### ClowdEnvironmentSpec



ClowdEnvironmentSpec defines the desired state of ClowdEnvironment.

_Appears in:_
- [ClowdEnvironment](#clowdenvironment)

| Field | Description |
| --- | --- |
| `targetNamespace` _string_ | TargetNamespace describes the namespace where any generated environmental resources should end up, this is particularly important in (*_local_*) mode. |
| `providers` _[ProvidersConfig](#providersconfig)_ | A ProvidersConfig object, detailing the setup and configuration of all the providers used in this ClowdEnvironment. |
| `resourceDefaults` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#resourcerequirements-v1-core)_ | Defines the default resource requirements in standard k8s format in the event that they omitted from a PodSpec inside a ClowdApp. |
| `serviceConfig` _[ServiceConfig](#serviceconfig)_ |  |
| `disabled` _boolean_ | Disabled turns off reconciliation for this ClowdEnv |




#### ClowdJobInvocation



ClowdJobInvocation is the Schema for the jobinvocations API

_Appears in:_
- [ClowdJobInvocationList](#clowdjobinvocationlist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloud.redhat.com/v1alpha1`
| `kind` _string_ | `ClowdJobInvocation`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ClowdJobInvocationSpec](#clowdjobinvocationspec)_ |  |


#### ClowdJobInvocationList



ClowdJobInvocationList contains a list of ClowdJobInvocation



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloud.redhat.com/v1alpha1`
| `kind` _string_ | `ClowdJobInvocationList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[ClowdJobInvocation](#clowdjobinvocation) array_ |  |


#### ClowdJobInvocationSpec



ClowdJobInvocationSpec defines the desired state of ClowdJobInvocation

_Appears in:_
- [ClowdJobInvocation](#clowdjobinvocation)

| Field | Description |
| --- | --- |
| `appName` _string_ | Name of the ClowdApp who owns the jobs |
| `jobs` _string array_ | Jobs is the set of jobs to be run by the invocation |
| `testing` _[JobTestingSpec](#jobtestingspec)_ | Testing is the struct for building out test jobs (iqe, etc) in a CJI |
| `runOnNotReady` _boolean_ | RunOnNotReady is a flag that when true, the job will not wait for the deployment to be ready to run |




#### ConfigAccessMode

_Underlying type:_ _string_

Describes what amount of app config is mounted to the pod

_Appears in:_
- [TestingConfig](#testingconfig)



#### CyndiSpec



CyndiSpec is used to indicate whether a ClowdApp needs database syndication configured by the cyndi operator and exposes a limited set of cyndi configuration options

_Appears in:_
- [ClowdAppSpec](#clowdappspec)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | Enables or Disables the Cyndi pipeline for the Clowdapp |
| `appName` _string_ | Application name - if empty will default to Clowdapp's name |
| `additionalFilters` _object array_ | AdditionalFilters |
| `insightsOnly` _boolean_ | Desired host syndication type (all or Insights hosts only) - defaults to false (All hosts) |


#### DatabaseConfig



DatabaseConfig configures the Clowder provider controlling the creation of Database instances.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[DatabaseMode](#databasemode)_ | The mode of operation of the Clowder Database Provider. Valid options are: (*_app-interface_*) where the provider will pass through database credentials found in the secret defined by the database name in the ClowdApp, and (*_local_*) where the provider will spin up a local instance of the database. |
| `caBundleURL` _string_ | Indicates where Clowder will fetch the database CA certificate bundle from. Currently only used in (*_app-interface_*) mode. If none is specified, the AWS RDS combined CA bundle is used. |
| `pvc` _boolean_ | If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes. |


#### DatabaseMode

_Underlying type:_ _string_

DatabaseMode details the mode of operation of the Clowder Database Provider

_Appears in:_
- [DatabaseConfig](#databaseconfig)



#### DatabaseSpec



DatabaseSpec is a struct defining a database to be exposed to a ClowdApp.

_Appears in:_
- [ClowdAppSpec](#clowdappspec)

| Field | Description |
| --- | --- |
| `version` _integer_ | Defines the Version of the PostGreSQL database, defaults to 12. |
| `name` _string_ | Defines the Name of the database used by this app. This will be used as the name of the logical database created by Clowder when the DB provider is in (*_local_*) mode. In (*_app-interface_*) mode, the name here is used to locate the DB secret as a fallback mechanism in cases where there is no 'clowder/database: <app-name>' annotation set on any secrets by looking for a secret with 'db.host' starting with '<name>-<env>' where env is usually 'stage' or 'prod' |
| `sharedDbAppName` _string_ | Defines the Name of the app to share a database from |
| `dbVolumeSize` _string_ | T-shirt size, one of small, medium, large |
| `dbResourceSize` _string_ | T-shirt size, one of small, medium, large |


#### Deployment



Deployment defines a service running inside a ClowdApp and will output a deployment resource. Only one container per pod is allowed and this is defined in the PodSpec attribute.

_Appears in:_
- [ClowdAppSpec](#clowdappspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name defines the identifier of a Pod inside the ClowdApp. This name will be used along side the name of the ClowdApp itself to form a <app>-<pod> pattern which will be used for all other created resources and also for some labels. It must be unique within a ClowdApp. |
| `minReplicas` _integer_ | Deprecated: Use Replicas instead If Replicas is not set and MinReplicas is set, then MinReplicas will be used |
| `replicas` _integer_ | Defines the desired replica count for the pod |
| `web` _[WebDeprecated](#webdeprecated)_ | If set to true, creates a service on the webPort defined in the ClowdEnvironment resource, along with the relevant liveness and readiness probes. |
| `webServices` _[WebServices](#webservices)_ |  |
| `podSpec` _[PodSpec](#podspec)_ | PodSpec defines a container running inside a ClowdApp. |
| `k8sAccessLevel` _[K8sAccessLevel](#k8saccesslevel)_ | K8sAccessLevel defines the level of access for this deployment |
| `autoScaler` _[AutoScaler](#autoscaler)_ | AutoScaler defines the configuration for the Keda auto scaler |
| `autoScalerSimple` _[AutoScalerSimple](#autoscalersimple)_ |  |
| `deploymentStrategy` _[DeploymentStrategy](#deploymentstrategy)_ | DeploymentStrategy allows the deployment strategy to be set only if the deployment has no public service enabled |
| `metadata` _[DeploymentMetadata](#deploymentmetadata)_ | Refer to Kubernetes API documentation for fields of `metadata`. |


#### DeploymentConfig





_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `omitPullPolicy` _boolean_ |  |


#### DeploymentInfo



DeploymentInfo defailts information about a specific deployment.

_Appears in:_
- [AppInfo](#appinfo)

| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `hostname` _string_ |  |
| `port` _integer_ |  |


#### DeploymentMetadata





_Appears in:_
- [Deployment](#deployment)

| Field | Description |
| --- | --- |
| `annotations` _object (keys:string, values:string)_ |  |


#### DeploymentStrategy





_Appears in:_
- [Deployment](#deployment)

| Field | Description |
| --- | --- |
| `privateStrategy` _[DeploymentStrategyType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#deploymentstrategytype-v1-apps)_ | PrivateStrategy allows a deployment that only uses a private port to set the deployment strategy one of Recreate or Rolling, default for a private service is Recreate. This is to enable a quicker roll out for services that do not have public facing endpoints. |


#### EnvResourceStatus





_Appears in:_
- [ClowdEnvironmentStatus](#clowdenvironmentstatus)

| Field | Description |
| --- | --- |
| `managedDeployments` _integer_ |  |
| `readyDeployments` _integer_ |  |
| `managedTopics` _integer_ |  |
| `readyTopics` _integer_ |  |


#### FeatureFlagsConfig



FeatureFlagsConfig configures the Clowder provider controlling the creation of FeatureFlag instances.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[FeatureFlagsMode](#featureflagsmode)_ | The mode of operation of the Clowder FeatureFlag Provider. Valid options are: (*_app-interface_*) where the provider will pass through credentials to the app configuration, and (*_local_*) where a local Unleash instance will be created. |
| `pvc` _boolean_ | If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes. |
| `credentialRef` _[NamespacedName](#namespacedname)_ | Defines the secret containing the client access token, only used for (*_app-interface_*) mode. |
| `hostname` _string_ | Defines the hostname for (*_app-interface_*) mode |
| `port` _integer_ | Defineds the port for (*_app-interface_*) mode |


#### FeatureFlagsMode

_Underlying type:_ _string_

FeatureFlagsMode details the mode of operation of the Clowder FeatureFlags Provider

_Appears in:_
- [FeatureFlagsConfig](#featureflagsconfig)



#### GatewayCert





_Appears in:_
- [WebConfig](#webconfig)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | Determines whether to enable the gateway cert, default is disabled |
| `certMode` _[GatewayCertMode](#gatewaycertmode)_ | Determines the mode of certificate generation, either self-signed or acme |
| `localCAConfigMap` _string_ | Determines a ConfigMap in the target namespace of the env which has ca.pem detailing the cert to use for mTLS verification |
| `emailAddress` _string_ | The email address used to register with Let's Encrypt for acme mode certs |


#### GatewayCertMode

_Underlying type:_ _string_

GatewayCertMode details the mode of operation of the Gateway Cert

_Appears in:_
- [GatewayCert](#gatewaycert)



#### InMemoryDBConfig



InMemoryDBConfig configures the Clowder provider controlling the creation of InMemoryDB instances.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[InMemoryMode](#inmemorymode)_ | The mode of operation of the Clowder InMemory Provider. Valid options are: (*_redis_*) where a local Minio instance will be created, and (*_elasticache_*) which will search the namespace of the ClowdApp for a secret called 'elasticache' |
| `pvc` _boolean_ | If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes. |


#### InMemoryMode

_Underlying type:_ _string_

InMemoryMode details the mode of operation of the Clowder InMemoryDB Provider

_Appears in:_
- [InMemoryDBConfig](#inmemorydbconfig)



#### InitContainer



InitContainer is a struct defining a k8s init container. This will be deployed along with the parent pod and is used to carry out one time initialization procedures.

_Appears in:_
- [PodSpec](#podspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name gives an identifier in the situation where multiple init containers exist |
| `image` _string_ | Image refers to the container image used to create the init container (if different from the primary pod image). |
| `command` _string array_ | A list of commands to run inside the parent Pod. |
| `args` _string array_ | A list of args to be passed to the init container. |
| `inheritEnv` _boolean_ | If true, inheirts the environment variables from the parent pod. specification |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#envvar-v1-core) array_ | A list of environment variables used only by the initContainer. |


#### IqeConfig





_Appears in:_
- [TestingConfig](#testingconfig)

| Field | Description |
| --- | --- |
| `imageBase` _string_ |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#resourcerequirements-v1-core)_ | A pass-through of a resource requirements in k8s ResourceRequirements format. If omitted, the default resource requirements from the ClowdEnvironment will be used. |
| `vaultSecretRef` _[NamespacedName](#namespacedname)_ | Defines the secret reference for loading vault credentials into the IQE job |
| `ui` _[IqeUIConfig](#iqeuiconfig)_ | Defines configurations related to UI testing containers |


#### IqeJobSpec





_Appears in:_
- [JobTestingSpec](#jobtestingspec)

| Field | Description |
| --- | --- |
| `imageTag` _string_ | Image tag to use for IQE container. By default, Clowder will set the image tag to be baseImage:name-of-iqe-plugin, where baseImage is defined in the ClowdEnvironment. Only the tag can be overridden here. |
| `plugins` _string_ | A comma,separated,list indicating IQE plugin(s) to run tests for. By default, Clowder will use the plugin name given on the ClowdApp's spec.testing.iqePlugin field. Use this field if you wish you override the plugin list. |
| `ui` _[IqeUISpec](#iqeuispec)_ | Defines configuration for a selenium container (optional) |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#envvar-v1-core)_ | Specifies environment variables to set on the IQE container |
| `debug` _boolean_ | Changes entrypoint to invoke 'iqe container-debug' so that container starts but does not run tests, allowing 'rsh' to be invoked |
| `marker` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_MARKER_EXPRESSION env var on the IQE container |
| `dynaconfEnvName` _string_ | (DEPRECATED, using 'env' now preferred) sets ENV_FOR_DYNACONF env var on the IQE container |
| `filter` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_FILTER_EXPRESSION env var on the IQE container |
| `requirements` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_REQUIREMENTS env var on the IQE container |
| `requirementsPriority` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_REQUIREMENTS_PRIORITY env var on the IQE container |
| `testImportance` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_TEST_IMPORTANCE env var on the IQE container |
| `logLevel` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_LOG_LEVEL env var on the IQE container |
| `parallelEnabled` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_PARALLEL_ENABLED env var on the IQE container |
| `parallelWorkerCount` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_PARALLEL_WORKER_COUNT env var on the IQE container |
| `rpArgs` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_RP_ARGS env var on the IQE container |
| `ibutsuSource` _string_ | (DEPRECATED, using 'env' now preferred) sets IQE_IBUTSU_SOURCE env var on the IQE container |


#### IqeSeleniumSpec





_Appears in:_
- [IqeUISpec](#iqeuispec)

| Field | Description |
| --- | --- |
| `deploy` _boolean_ | Whether or not a selenium container should be deployed in the IQE pod |
| `imageTag` _string_ | Name of selenium image tag to use if not using the environment's default |


#### IqeUIConfig





_Appears in:_
- [IqeConfig](#iqeconfig)

| Field | Description |
| --- | --- |
| `selenium` _[IqeUISeleniumConfig](#iqeuiseleniumconfig)_ | Defines configurations for selenium containers in this environment |


#### IqeUISeleniumConfig





_Appears in:_
- [IqeUIConfig](#iqeuiconfig)

| Field | Description |
| --- | --- |
| `imageBase` _string_ | Defines the image used for selenium containers in this environment |
| `defaultImageTag` _string_ | Defines the default image tag used for selenium containers in this environment |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#resourcerequirements-v1-core)_ | Defines the resource requests/limits set on selenium containers |


#### IqeUISpec





_Appears in:_
- [IqeJobSpec](#iqejobspec)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | No longer used |
| `selenium` _[IqeSeleniumSpec](#iqeseleniumspec)_ | Configuration options for running IQE with a selenium container |


#### Job



Job defines a ClowdJob A Job struct will deploy as a CronJob if `schedule` is set and will deploy as a Job if it is not set. Unsupported fields will be dropped from Jobs

_Appears in:_
- [ClowdAppSpec](#clowdappspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name defines identifier of the Job. This name will be used to name the CronJob resource, the container will be name identically. |
| `disabled` _boolean_ | Disabled allows a job to be disabled, as such, the resource is not created on the system and cannot be invoked with a CJI |
| `schedule` _string_ | Defines the schedule for the job to run |
| `parallelism` _integer_ | Defines the parallelism of the job |
| `completions` _integer_ | Defines the completions of the job |
| `podSpec` _[PodSpec](#podspec)_ | PodSpec defines a container running inside the CronJob. |
| `restartPolicy` _[RestartPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#restartpolicy-v1-core)_ | Defines the restart policy for the CronJob, defaults to never |
| `concurrencyPolicy` _[ConcurrencyPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#concurrencypolicy-v1-batch)_ | Defines the concurrency policy for the CronJob, defaults to Allow Only applies to Cronjobs |
| `suspend` _boolean_ | This flag tells the controller to suspend subsequent executions, it does not apply to already started executions.  Defaults to false. Only applies to Cronjobs |
| `successfulJobsHistoryLimit` _integer_ | The number of successful finished jobs to retain. Value must be non-negative integer. Defaults to 3. Only applies to Cronjobs |
| `failedJobsHistoryLimit` _integer_ | The number of failed finished jobs to retain. Value must be non-negative integer. Defaults to 1. Only applies to Cronjobs |
| `startingDeadlineSeconds` _integer_ | Defines the StartingDeadlineSeconds for the CronJob |
| `activeDeadlineSeconds` _integer_ | The activeDeadlineSeconds for the Job or CronJob. More info: https://kubernetes.io/docs/concepts/workloads/controllers/job/ |


#### JobConditionState

_Underlying type:_ _string_



_Appears in:_
- [ClowdJobInvocationStatus](#clowdjobinvocationstatus)



#### JobTestingSpec





_Appears in:_
- [ClowdJobInvocationSpec](#clowdjobinvocationspec)

| Field | Description |
| --- | --- |
| `iqe` _[IqeJobSpec](#iqejobspec)_ | Iqe is the job spec to override defaults from the ClowdApp's definition of the job |


#### K8sAccessLevel

_Underlying type:_ _string_

K8sAccessLevel defines the access level for the deployment, one of 'default', 'view' or 'edit'

_Appears in:_
- [Deployment](#deployment)
- [TestingConfig](#testingconfig)



#### KafkaClusterConfig



KafkaClusterConfig defines options related to the Kafka cluster managed/monitored by Clowder

_Appears in:_
- [KafkaConfig](#kafkaconfig)

| Field | Description |
| --- | --- |
| `name` _string_ | Defines the kafka cluster name (default: <ClowdEnvironment Name>-<UID>) |
| `namespace` _string_ | The namespace the kafka cluster is expected to reside in (default: the environment's targetNamespace) |
| `forceTLS` _boolean_ | Force TLS |
| `replicas` _integer_ | The requested number of replicas for kafka/zookeeper. If unset, default is '1' |
| `storageSize` _string_ | Persistent volume storage size. If unset, default is '1Gi' Only applies when KafkaConfig.PVC is set to 'true' |
| `deleteClaim` _boolean_ | Delete persistent volume claim if the Kafka cluster is deleted Only applies when KafkaConfig.PVC is set to 'true' |
| `version` _string_ | Version. If unset, default is '2.5.0' |
| `config` _map[string]string_ | Config full options |
| `jvmOptions` _[KafkaSpecKafkaJvmOptions](#kafkaspeckafkajvmoptions)_ | JVM Options |
| `resources` _[KafkaSpecKafkaResources](#kafkaspeckafkaresources)_ | Resource Limits |


#### KafkaConfig



KafkaConfig configures the Clowder provider controlling the creation of Kafka instances.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[KafkaMode](#kafkamode)_ | The mode of operation of the Clowder Kafka Provider. Valid options are: (*_operator_*) which provisions Strimzi resources and will configure KafkaTopic CRs and place them in the Kafka cluster's namespace described in the configuration, (*_app-interface_*) which simply passes the topic names through to the App's cdappconfig.json and expects app-interface to have created the relevant topics, and (*_local_*) where a small instance of Kafka is created in the desired cluster namespace and configured to auto-create topics. |
| `enableLegacyStrimzi` _boolean_ | EnableLegacyStrimzi disables TLS + user auth |
| `pvc` _boolean_ | If using the (*_local_*) or (*_operator_*) mode and PVC is set to true, this sets the provisioned Kafka instance to use a PVC instead of emptyDir for its volumes. |
| `cluster` _[KafkaClusterConfig](#kafkaclusterconfig)_ | Defines options related to the Kafka cluster for this environment. Ignored for (*_local_*) mode. |
| `connect` _[KafkaConnectClusterConfig](#kafkaconnectclusterconfig)_ | Defines options related to the Kafka Connect cluster for this environment. Ignored for (*_local_*) mode. |
| `managedSecretRef` _[NamespacedName](#namespacedname)_ | Defines the secret reference for the Managed Kafka mode. Only used in (*_managed_*) mode. |
| `managedPrefix` _string_ | Managed topic prefix for the managed cluster. Only used in (*_managed_*) mode. |
| `topicNamespace` _string_ | Namespace that kafkaTopics should be written to for (*_msk_*) mode. |
| `clusterAnnotation` _string_ | Cluster annotation identifier for (*_msk_*) mode. |
| `clusterName` _string_ | (Deprecated) Defines the cluster name to be used by the Kafka Provider this will be used in some modes to locate the Kafka instance. |
| `namespace` _string_ | (Deprecated) The Namespace the cluster is expected to reside in. This is only used in (*_app-interface_*) and (*_operator_*) modes. |
| `connectNamespace` _string_ | (Deprecated) The namespace that the Kafka Connect cluster is expected to reside in. This is only used in (*_app-interface_*) and (*_operator_*) modes. |
| `connectClusterName` _string_ | (Deprecated) Defines the kafka connect cluster name that is used in this environment. |
| `suffix` _string_ | (Deprecated) (Unused) |
| `kafkaConnectReplicaCount` _integer_ | Sets the replica count for ephem-msk mode for kafka connect |


#### KafkaConnectClusterConfig



KafkaConnectClusterConfig defines options related to the Kafka Connect cluster managed/monitored by Clowder

_Appears in:_
- [KafkaConfig](#kafkaconfig)

| Field | Description |
| --- | --- |
| `name` _string_ | Defines the kafka connect cluster name (default: <kafka cluster's name>) |
| `namespace` _string_ | The namespace the kafka connect cluster is expected to reside in (default: the kafka cluster's namespace) |
| `replicas` _integer_ | The requested number of replicas for kafka connect. If unset, default is '1' |
| `version` _string_ | Version. If unset, default is '2.5.0' |
| `image` _string_ | Image. If unset, default is 'quay.io/cloudservices/xjoin-kafka-connect-strimzi:latest' |
| `resources` _[KafkaConnectSpecResources](#kafkaconnectspecresources)_ | Resource Limits |


#### KafkaMode

_Underlying type:_ _string_

KafkaMode details the mode of operation of the Clowder Kafka Provider

_Appears in:_
- [KafkaConfig](#kafkaconfig)



#### KafkaTopicSpec



KafkaTopicSpec defines the desired state of KafkaTopic

_Appears in:_
- [ClowdAppSpec](#clowdappspec)

| Field | Description |
| --- | --- |
| `config` _object (keys:string, values:string)_ | A key/value pair describing the configuration of a particular topic. |
| `partitions` _integer_ | The requested number of partitions for this topic. If unset, default is '3' |
| `replicas` _integer_ | The requested number of replicas for this topic. If unset, default is '3' |
| `topicName` _string_ | The requested name for this topic. |


#### LoggingConfig



LoggingConfig configures the Clowder provider controlling the creation of Logging instances.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[LoggingMode](#loggingmode)_ | The mode of operation of the Clowder Logging Provider. Valid options are: (*_app-interface_*) where the provider will pass through cloudwatch credentials to the app configuration, and (*_none_*) where no logging will be configured. |


#### LoggingMode

_Underlying type:_ _string_

LoggingMode details the mode of operation of the Clowder Logging Provider

_Appears in:_
- [LoggingConfig](#loggingconfig)



#### MetricsConfig



MetricsConfig configures the Clowder provider controlling the creation of metrics services and their probes.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `port` _integer_ | The port that metrics services inside ClowdApp pods should be served on. |
| `path` _string_ | A prefix path that pods will be instructed to use when setting up their metrics server. |
| `mode` _[MetricsMode](#metricsmode)_ | The mode of operation of the Metrics provider. The allowed modes are (*_none_*), which disables metrics service generation, or (*_operator_*) where services and probes are generated. (*_app-interface_*) where services and probes are generated for app-interface. |
| `prometheus` _[PrometheusConfig](#prometheusconfig)_ | Prometheus specific configuration |


#### MetricsMode

_Underlying type:_ _string_

MetricsMode details the mode of operation of the Clowder Metrics Provider

_Appears in:_
- [MetricsConfig](#metricsconfig)



#### MetricsWebService



MetricsWebService is the definition of the metrics web service. This is automatically enabled and the configuration here at the moment is included for completeness, as there are no configurable options.

_Appears in:_
- [WebServices](#webservices)





#### NamespacedName



NamespacedName type to represent a real Namespaced Name

_Appears in:_
- [FeatureFlagsConfig](#featureflagsconfig)
- [IqeConfig](#iqeconfig)
- [KafkaConfig](#kafkaconfig)
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `name` _string_ | Name defines the Name of a resource. |
| `namespace` _string_ | Namespace defines the Namespace of a resource. |


#### ObjectStoreConfig



ObjectStoreConfig configures the Clowder provider controlling the creation of ObjectStore instances.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[ObjectStoreMode](#objectstoremode)_ | The mode of operation of the Clowder ObjectStore Provider. Valid options are: (*_app-interface_*) where the provider will pass through Amazon S3 credentials to the app configuration, and (*_minio_*) where a local Minio instance will be created. |
| `suffix` _string_ | Currently unused. |
| `pvc` _boolean_ | If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes. |


#### ObjectStoreMode

_Underlying type:_ _string_

ObjectStoreMode details the mode of operation of the Clowder ObjectStore Provider

_Appears in:_
- [ObjectStoreConfig](#objectstoreconfig)



#### PodSpec



PodSpec defines a container running inside a ClowdApp.

_Appears in:_
- [Deployment](#deployment)
- [Job](#job)

| Field | Description |
| --- | --- |
| `image` _string_ | Image refers to the container image used to create the pod. |
| `initContainers` _[InitContainer](#initcontainer) array_ | A list of init containers used to perform at-startup operations. |
| `metadata` _[PodspecMetadata](#podspecmetadata)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `command` _string array_ | The command that will be invoked inside the pod at startup. |
| `args` _string array_ | A list of args to be passed to the pod container. |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#envvar-v1-core) array_ | A list of environment variables in k8s defined format. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#resourcerequirements-v1-core)_ | A pass-through of a resource requirements in k8s ResourceRequirements format. If omitted, the default resource requirements from the ClowdEnvironment will be used. |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#probe-v1-core)_ | A pass-through of a Liveness Probe specification in standard k8s format. If omitted, a standard probe will be setup point to the webPort defined in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to false. |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#probe-v1-core)_ | A pass-through of a Readiness Probe specification in standard k8s format. If omitted, a standard probe will be setup point to the webPort defined in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to false. |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#volume-v1-core) array_ | A pass-through of a list of Volumes in standa k8s format. |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#volumemount-v1-core) array_ | A pass-through of a list of VolumesMounts in standa k8s format. |
| `lifecycle` _[Lifecycle](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#lifecycle-v1-core)_ | A pass-through of Lifecycle specification in standard k8s format |
| `terminationGracePeriodSeconds` _integer_ | A pass-through of TerminationGracePeriodSeconds specification in standard k8s format default is 30 seconds |
| `sidecars` _[Sidecar](#sidecar) array_ | Lists the expected side cars, will be validated in the validating webhook |
| `machinePool` _string_ | MachinePool allows the pod to be scheduled to a particular machine pool. |


#### PodspecMetadata



Metadata for applying annotations etc to PodSpec

_Appears in:_
- [PodSpec](#podspec)

| Field | Description |
| --- | --- |
| `annotations` _object (keys:string, values:string)_ |  |


#### PrivateWebService



PrivateWebService is the definition of the private web service. There can be only one private service managed by Clowder.

_Appears in:_
- [WebServices](#webservices)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | Enabled describes if Clowder should enable the private service and provide the configuration in the cdappconfig. |
| `appProtocol` _[AppProtocol](#appprotocol)_ | AppProtocol determines the protocol to be used for the private port, (defaults to http) |


#### PrometheusConfig





_Appears in:_
- [MetricsConfig](#metricsconfig)

| Field | Description |
| --- | --- |
| `deploy` _boolean_ | Determines whether to deploy prometheus in operator mode |
| `appInterfaceHostname` _string_ | Specify prometheus hostname when in app-interface mode |


#### PrometheusStatus



PrometheusStatus provides info on how to connect to Prometheus

_Appears in:_
- [ClowdEnvironmentStatus](#clowdenvironmentstatus)

| Field | Description |
| --- | --- |
| `hostname` _string_ |  |


#### ProvidersConfig



ProvidersConfig defines a group of providers configuration for a ClowdEnvironment.

_Appears in:_
- [ClowdEnvironmentSpec](#clowdenvironmentspec)

| Field | Description |
| --- | --- |
| `db` _[DatabaseConfig](#databaseconfig)_ | Defines the Configuration for the Clowder Database Provider. |
| `inMemoryDb` _[InMemoryDBConfig](#inmemorydbconfig)_ | Defines the Configuration for the Clowder InMemoryDB Provider. |
| `kafka` _[KafkaConfig](#kafkaconfig)_ | Defines the Configuration for the Clowder Kafka Provider. |
| `logging` _[LoggingConfig](#loggingconfig)_ | Defines the Configuration for the Clowder Logging Provider. |
| `metrics` _[MetricsConfig](#metricsconfig)_ | Defines the Configuration for the Clowder Metrics Provider. |
| `objectStore` _[ObjectStoreConfig](#objectstoreconfig)_ | Defines the Configuration for the Clowder ObjectStore Provider. |
| `web` _[WebConfig](#webconfig)_ | Defines the Configuration for the Clowder Web Provider. |
| `featureFlags` _[FeatureFlagsConfig](#featureflagsconfig)_ | Defines the Configuration for the Clowder FeatureFlags Provider. |
| `serviceMesh` _[ServiceMeshConfig](#servicemeshconfig)_ | Defines the Configuration for the Clowder ServiceMesh Provider. |
| `pullSecrets` _[NamespacedName](#namespacedname) array_ | Defines the pull secret to use for the service accounts. |
| `testing` _[TestingConfig](#testingconfig)_ | Defines the environment for iqe/smoke testing |
| `sidecars` _[Sidecars](#sidecars)_ | Defines the sidecar configuration |
| `autoScaler` _[AutoScalerConfig](#autoscalerconfig)_ | Defines the autoscaler configuration |
| `deployment` _[DeploymentConfig](#deploymentconfig)_ | Defines the Deployment provider options |


#### PublicWebService



PublicWebService is the definition of the public web service. There can be only one public service managed by Clowder.

_Appears in:_
- [WebServices](#webservices)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | Enabled describes if Clowder should enable the public service and provide the configuration in the cdappconfig. |
| `apiPath` _string_ | (DEPRECATED, use apiPaths instead) Configures a path named '/api/<apiPath>/' that this app will serve requests from. |
| `apiPaths` _[APIPath](#apipath) array_ | Defines a list of API paths (each matching format: "/api/some-path/") that this app will serve requests from. |
| `whitelistPaths` _string array_ | WhitelistPaths define the paths that do not require authentication |


#### ServiceConfig



ServiceConfig provides options for k8s Service resources

_Appears in:_
- [ClowdEnvironmentSpec](#clowdenvironmentspec)

| Field | Description |
| --- | --- |
| `type` _string_ |  |


#### ServiceMeshConfig



ServiceMeshConfig determines if this env should be part of a service mesh and, if enabled, configures the service mesh

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `mode` _[ServiceMeshMode](#servicemeshmode)_ |  |


#### ServiceMeshMode

_Underlying type:_ _string_

ServiceMeshMode just determines if we enable or disable the service mesh

_Appears in:_
- [ServiceMeshConfig](#servicemeshconfig)



#### Sidecar





_Appears in:_
- [PodSpec](#podspec)

| Field | Description |
| --- | --- |
| `name` _string_ | The name of the sidecar, only supported names allowed, (token-refresher) |
| `enabled` _boolean_ | Defines if the sidecar is enabled, defaults to False |


#### Sidecars





_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `tokenRefresher` _[TokenRefresherConfig](#tokenrefresherconfig)_ | Sets up Token Refresher configuration |


#### SimpleAutoScalerMetric



SimpleAutoScalerMetric defines a metric of either a value or utilization

_Appears in:_
- [AutoScalerSimple](#autoscalersimple)

| Field | Description |
| --- | --- |
| `scaleAtValue` _string_ |  |
| `scaleAtUtilization` _integer_ |  |


#### SimpleAutoScalerReplicas



SimpleAutoScalerReplicas defines the minimum and maximum replica counts for the auto scaler

_Appears in:_
- [AutoScalerSimple](#autoscalersimple)

| Field | Description |
| --- | --- |
| `min` _integer_ |  |
| `max` _integer_ |  |


#### TLS





_Appears in:_
- [WebConfig](#webconfig)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ |  |
| `port` _integer_ |  |
| `privatePort` _integer_ |  |


#### TestingConfig





_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `iqe` _[IqeConfig](#iqeconfig)_ | Defines the environment for iqe/smoke testing |
| `k8sAccessLevel` _[K8sAccessLevel](#k8saccesslevel)_ | The mode of operation of the testing Pod. Valid options are: 'default', 'view' or 'edit' |
| `configAccess` _[ConfigAccessMode](#configaccessmode)_ | The mode of operation for access to outside app configs. Valid options are: (*_none_*) -- no app config is mounted to the pod (*_app_*) -- only the ClowdApp's config is mounted to the pod (*_environment_*) -- the config for all apps in the env are mounted |


#### TestingSpec





_Appears in:_
- [ClowdAppSpec](#clowdappspec)

| Field | Description |
| --- | --- |
| `iqePlugin` _string_ |  |


#### TokenRefresherConfig





_Appears in:_
- [Sidecars](#sidecars)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | Enables or disables token refresher sidecars |


#### WebConfig



WebConfig configures the Clowder provider controlling the creation of web services and their probes.

_Appears in:_
- [ProvidersConfig](#providersconfig)

| Field | Description |
| --- | --- |
| `port` _integer_ | The port that web services inside ClowdApp pods should be served on. |
| `privatePort` _integer_ | The private port that web services inside a ClowdApp should be served on. |
| `aiuthPort` _integer_ | The auth port that the web local mode will use with the AuthSidecar |
| `apiPrefix` _string_ | An api prefix path that pods will be instructed to use when setting up their web server. |
| `mode` _[WebMode](#webmode)_ | The mode of operation of the Web provider. The allowed modes are (*_none_*/*_operator_*), and (*_local_*) which deploys keycloak and BOP. |
| `bopURL` _string_ | The URL of BOP - only used in (*_none_*/*_operator_*) mode. |
| `ingressClass` _string_ | Ingress Class Name used only in (*_local_*) mode. |
| `keycloakVersion` _string_ | Optional keycloak version override -- used only in (*_local_*) mode -- if not set, a hard-coded default is used. |
| `keycloakPVC` _boolean_ | Optionally use PVC storage for keycloak db |
| `images` _[WebImages](#webimages)_ | Optional images to use for web provider components -- only applies when running in (*_local_*) mode. |
| `tls` _[TLS](#tls)_ | TLS sidecar enablement |
| `gatewayCert` _[GatewayCert](#gatewaycert)_ | Gateway cert |


#### WebDeprecated

_Underlying type:_ _boolean_

WebDeprecated defines a boolean flag to help distinguish from the newer WebServices

_Appears in:_
- [Deployment](#deployment)



#### WebImages



WebImages defines optional container image overrides for the web provider components

_Appears in:_
- [WebConfig](#webconfig)

| Field | Description |
| --- | --- |
| `mocktitlements` _string_ | Mock entitlements image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used. |
| `keycloak` _string_ | Keycloak image -- default is 'quay.io/keycloak/keycloak:{KeycloakVersion}' unless overridden here |
| `caddy` _string_ | Caddy image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used. |
| `caddyGateway` _string_ | Caddy Gateway image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used. |
| `mockBop` _string_ | Mock BOP image -- if not defined, value from operator config is used if set, otherwise a hard-coded default is used. |


#### WebMode

_Underlying type:_ _string_

WebMode details the mode of operation of the Clowder Web Provider

_Appears in:_
- [WebConfig](#webconfig)



#### WebServices



WebServices defines the structs for the three exposed web services: public, private and metrics.

_Appears in:_
- [Deployment](#deployment)

| Field | Description |
| --- | --- |
| `public` _[PublicWebService](#publicwebservice)_ |  |
| `private` _[PrivateWebService](#privatewebservice)_ |  |
| `metrics` _[MetricsWebService](#metricswebservice)_ |  |


