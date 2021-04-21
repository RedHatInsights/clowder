API Reference
=============

Packages
--------

- :ref:`cloud.redhat.com/v1alpha1`
- :ref:`cyndi.cloud.redhat.com/v1alpha1`


.. _cloud.redhat.com/v1alpha1:

cloud.redhat.com/v1alpha1
-------------------------

Package v1alpha1 contains API Schema definitions for the cloud.redhat.com v1alpha1 API group

Resource Types
**************

- :ref:`ClowdApp`
- :ref:`ClowdAppList`
- :ref:`ClowdEnvironment`
- :ref:`ClowdEnvironmentList`
- :ref:`ClowdJobInvocation`
- :ref:`ClowdJobInvocationList`



.. _AppInfo :

AppInfo 
^^^^^^^



Appears In:
:ref:`ClowdEnvironmentStatus`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``name`` (string)", ""
   "``deployments`` (:ref:`DeploymentInfo` array)", ""


.. _ClowdApp :

ClowdApp 
^^^^^^^^

ClowdApp is the Schema for the clowdapps API

Appears In:
:ref:`ClowdAppList`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`ClowdApp`"
   "``metadata`` (`ObjectMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``spec`` (:ref:`ClowdAppSpec`)", "A ClowdApp specification."


.. _ClowdAppList :

ClowdAppList 
^^^^^^^^^^^^

ClowdAppList contains a list of ClowdApp




.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`ClowdAppList`"
   "``metadata`` (`ListMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``items`` (:ref:`ClowdApp`)", "A list of ClowdApp Resources."


.. _ClowdAppSpec :

ClowdAppSpec 
^^^^^^^^^^^^

ClowdAppSpec is the main specification for a single Clowder Application it defines n pods along with dependencies that are shared between them.

Appears In:
:ref:`ClowdApp`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``deployments`` (:ref:`Deployment`)", "A list of deployments"
   "``jobs`` (:ref:`Job`)", "A list of jobs"
   "``pods`` (:ref:`PodSpecDeprecated`)", "Deprecated"
   "``envName`` (string)", "The name of the ClowdEnvironment resource that this ClowdApp will use as its base. This does not mean that the ClowdApp needs to be placed in the same directory as the targetNamespace of the ClowdEnvironment."
   "``kafkaTopics`` (:ref:`KafkaTopicSpec`)", "A list of Kafka topics that will be created and made available to all the pods listed in the ClowdApp."
   "``database`` (:ref:`DatabaseSpec`)", "The database specification defines a single database, the configuration of which will be made available to all the pods in the ClowdApp."
   "``objectStore`` (string array)", "A list of string names defining storage buckets. In certain modes, defined by the ClowdEnvironment, Clowder will create those buckets."
   "``inMemoryDb`` (boolean)", "If inMemoryDb is set to true, Clowder will pass configuration of an In Memory Database to the pods in the ClowdApp. This single instance will be shared between all apps."
   "``featureFlags`` (boolean)", "If featureFlags is set to true, Clowder will pass configuration of a FeatureFlags instance to the pods in the ClowdApp. This single instance will be shared between all apps."
   "``dependencies`` (string array)", "A list of dependencies in the form of the name of the ClowdApps that are required to be present for this ClowdApp to function."
   "``optionalDependencies`` (string array)", "A list of optional dependencies in the form of the name of the ClowdApps that are will be added to the configuration when present."
   "``cyndi`` (:ref:`CyndiSpec`)", "Configures 'cyndi' database syndication for this app. When the app's ClowdEnvironment has the kafka provider set to (*_operator_*) mode, Clowder will configure a CyndiPipeline for this app in the environment's kafka-connect namespace. When the kafka provider is in (*_app-interface_*) mode, Clowder will check to ensure that a CyndiPipeline resource exists for the application in the environment's kafka-connect namespace. For all other kafka provider modes, this configuration option has no effect."




.. _ClowdEnvironment :

ClowdEnvironment 
^^^^^^^^^^^^^^^^

ClowdEnvironment is the Schema for the clowdenvironments API

Appears In:
:ref:`ClowdEnvironmentList`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`ClowdEnvironment`"
   "``metadata`` (`ObjectMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``spec`` (:ref:`ClowdEnvironmentSpec`)", "A ClowdEnvironmentSpec object."


.. _ClowdEnvironmentList :

ClowdEnvironmentList 
^^^^^^^^^^^^^^^^^^^^

ClowdEnvironmentList contains a list of ClowdEnvironment




.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`ClowdEnvironmentList`"
   "``metadata`` (`ListMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``items`` (:ref:`ClowdEnvironment`)", "A list of ClowdEnvironment objects."


.. _ClowdEnvironmentSpec :

ClowdEnvironmentSpec 
^^^^^^^^^^^^^^^^^^^^

ClowdEnvironmentSpec defines the desired state of ClowdEnvironment.

Appears In:
:ref:`ClowdEnvironment`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``targetNamespace`` (string)", "TargetNamespace describes the namespace where any generated environmental resources should end up, this is particularly important in (*_local_*) mode."
   "``providers`` (:ref:`ProvidersConfig`)", "A ProvidersConfig object, detailing the setup and configuration of all the providers used in this ClowdEnvironment."
   "``resourceDefaults`` (`ResourceRequirements <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#resourcerequirements-v1-core>`_)", "Defines the default resource requirements in standard k8s format in the event that they omitted from a PodSpec inside a ClowdApp."




.. _ClowdJobInvocation :

ClowdJobInvocation 
^^^^^^^^^^^^^^^^^^

ClowdJobInvocation is the Schema for the jobinvocations API

Appears In:
:ref:`ClowdJobInvocationList`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`ClowdJobInvocation`"
   "``metadata`` (`ObjectMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``spec`` (:ref:`ClowdJobInvocationSpec`)", ""


.. _ClowdJobInvocationList :

ClowdJobInvocationList 
^^^^^^^^^^^^^^^^^^^^^^

ClowdJobInvocationList contains a list of ClowdJobInvocation




.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`ClowdJobInvocationList`"
   "``metadata`` (`ListMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``items`` (:ref:`ClowdJobInvocation`)", ""


.. _ClowdJobInvocationSpec :

ClowdJobInvocationSpec 
^^^^^^^^^^^^^^^^^^^^^^

ClowdJobInvocationSpec defines the desired state of ClowdJobInvocation

Appears In:
:ref:`ClowdJobInvocation`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``appName`` (string)", "Name of the ClowdApp who owns the jobs"
   "``jobs`` (string array)", "Jobs is the set of jobs to be run by the invocation"




.. _CyndiSpec :

CyndiSpec 
^^^^^^^^^

CyndiSpec is used to indicate whether a ClowdApp needs database syndication configured by the cyndi operator and exposes a limited set of cyndi configuration options

Appears In:
:ref:`ClowdAppSpec`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``enabled`` (boolean)", ""
   "``appName`` (string)", ""
   "``insightsOnly`` (boolean)", ""


.. _DatabaseConfig :

DatabaseConfig 
^^^^^^^^^^^^^^

DatabaseConfig configures the Clowder provider controlling the creation of Database instances.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``mode`` (DatabaseMode)", "The mode of operation of the Clowder Database Provider. Valid options are: (*_app-interface_*) where the provider will pass through database credentials found in the secret defined by the database name in the ClowdApp, and (*_local_*) where the provider will spin up a local instance of the database."
   "``pvc`` (boolean)", "If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes."


.. _DatabaseSpec :

DatabaseSpec 
^^^^^^^^^^^^

DatabaseSpec is a struct defining a database to be exposed to a ClowdApp.

Appears In:
:ref:`ClowdAppSpec`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``version`` (integer)", "Defines the Version of the PostGreSQL database, defaults to 12."
   "``name`` (string)", "Defines the Name of the database to be created. This will be used as the name of the logical database inside the database server in (*_local_*) mode and the name of the secret to be used for Database configuration in (*_app-interface_*) mode."
   "``sharedDbAppName`` (string)", "Defines the Name of the app to share a database from"


.. _Deployment :

Deployment 
^^^^^^^^^^

Deployment defines a service running inside a ClowdApp and will output a deployment resource. Only one container per pod is allowed and this is defined in the PodSpec attribute.

Appears In:
:ref:`ClowdAppSpec`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``name`` (string)", "Name defines the identifier of a Pod inside the ClowdApp. This name will be used along side the name of the ClowdApp itself to form a <app>-<pod> pattern which will be used for all other created resources and also for some labels. It must be unique within a ClowdApp."
   "``minReplicas`` (integer)", "Defines the minimum replica count for the pod."
   "``web`` (WebDeprecated)", "If set to true, creates a service on the webPort defined in the ClowdEnvironment resource, along with the relevant liveness and readiness probes."
   "``webServices`` (:ref:`WebServices`)", ""
   "``podSpec`` (:ref:`PodSpec`)", "PodSpec defines a container running inside a ClowdApp."
   "``k8sAccessLevel`` (K8sAccessLevel)", "K8sAccessLevel defines the level of access for this deployment"


.. _DeploymentInfo :

DeploymentInfo 
^^^^^^^^^^^^^^



Appears In:
:ref:`AppInfo`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``name`` (string)", ""
   "``hostname`` (string)", ""
   "``port`` (integer)", ""


.. _FeatureFlagsConfig :

FeatureFlagsConfig 
^^^^^^^^^^^^^^^^^^

FeatureFlagsConfig configures the Clowder provider controlling the creation of FeatureFlag instances.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``mode`` (FeatureFlagsMode)", "The mode of operation of the Clowder FeatureFlag Provider. Valid options are: (*_app-interface_*) where the provider will pass through credentials to the app configuration, and (*_local_*) where a local Unleash instance will be created."
   "``pvc`` (boolean)", "If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes."


.. _InMemoryDBConfig :

InMemoryDBConfig 
^^^^^^^^^^^^^^^^

InMemoryDBConfig configures the Clowder provider controlling the creation of InMemoryDB instances.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``mode`` (InMemoryMode)", "The mode of operation of the Clowder InMemory Provider. Valid options are: (*_redis_*) where a local Minio instance will be created, and (*_elasticache_*) which will search the namespace of the ClowdApp for a secret called 'elasticache'"
   "``pvc`` (boolean)", "If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes."


.. _InitContainer :

InitContainer 
^^^^^^^^^^^^^

InitContainer is a struct defining a k8s init container. This will be deployed along with the parent pod and is used to carry out one time initialization procedures.

Appears In:
:ref:`PodSpec`
:ref:`PodSpecDeprecated`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``command`` (string array)", "A list of commands to run inside the parent Pod."
   "``args`` (string array)", "A list of args to be passed to the init container."
   "``inheritEnv`` (boolean)", "If true, inheirts the environment variables from the parent pod. specification"
   "``env`` (`EnvVar <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#envvar-v1-core>`_ array)", "A list of environment variables used only by the initContainer."


.. _Job :

Job 
^^^

Job defines a CronJob as Schedule is required. In the future omitting the Schedule field will allow support for a standard Job resource.

Appears In:
:ref:`ClowdAppSpec`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``name`` (string)", "Name defines identifier of the Job. This name will be used to name the CronJob resource, the container will be name identically."
   "``schedule`` (string)", "Defines the schedule for the job to run"
   "``podSpec`` (:ref:`PodSpec`)", "PodSpec defines a container running inside the CronJob."
   "``restartPolicy`` (`RestartPolicy <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#restartpolicy-v1-core>`_)", "Defines the restart policy for the CronJob, defaults to never"
   "``concurrencyPolicy`` (`ConcurrencyPolicy <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#concurrencypolicy-v1beta1-batch>`_)", "Defines the concurrency policy for the CronJob, defaults to Allow"
   "``startingDeadlineSeconds`` (integer)", "Defines the StartingDeadlineSeconds for the CronJob"


.. _KafkaClusterConfig :

KafkaClusterConfig 
^^^^^^^^^^^^^^^^^^

KafkaClusterConfig defines options related to the Kafka cluster managed/monitored by Clowder

Appears In:
:ref:`KafkaConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``name`` (string)", "Defines the kafka cluster name"
   "``namespace`` (string)", "The namespace the kafka cluster is expected to reside in (default: the environment's targetNamespace)"
   "``replicas`` (integer)", "The requested number of replicas for kafka/zookeeper. If unset, default is '1'"
   "``storageSize`` (string)", "Persistent volume storage size. If unset, default is '1Gi' Only applies when KafkaConfig.PVC is set to 'true'"
   "``deleteClaim`` (boolean)", "Delete persistent volume claim if the Kafka cluster is deleted Only applies when KafkaConfig.PVC is set to 'true'"
   "``version`` (string)", "Version. If unset, default is '2.5.0'"


.. _KafkaConfig :

KafkaConfig 
^^^^^^^^^^^

KafkaConfig configures the Clowder provider controlling the creation of Kafka instances.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``mode`` (KafkaMode)", "The mode of operation of the Clowder Kafka Provider. Valid options are: (*_operator_*) which provisions Strimzi resources and will configure KafkaTopic CRs and place them in the Kafka cluster's namespace described in the configuration, (*_app-interface_*) which simply passes the topic names through to the App's cdappconfig.json and expects app-interface to have created the relevant topics, and (*_local_*) where a small instance of Kafka is created in the desired cluster namespace and configured to auto-create topics."
   "``enableLegacyStrimzi`` (boolean)", "EnableLegacyStrimzi disables TLS + user auth"
   "``pvc`` (boolean)", "If using the (*_local_*) or (*_operator_*) mode and PVC is set to true, this sets the provisioned Kafka instance to use a PVC instead of emptyDir for its volumes."
   "``cluster`` (:ref:`KafkaClusterConfig`)", "Defines options related to the Kafka cluster for this environment. Ignored for (*_local_*) mode."
   "``connect`` (:ref:`KafkaConnectClusterConfig`)", "Defines options related to the Kafka Connect cluster for this environment. Ignored for (*_local_*) mode."
   "``clusterName`` (string)", "(Deprecated) Defines the cluster name to be used by the Kafka Provider this will be used in some modes to locate the Kafka instance."
   "``namespace`` (string)", "(Deprecated) The Namespace the cluster is expected to reside in. This is only used in (*_app-interface_*) and (*_operator_*) modes."
   "``connectNamespace`` (string)", "(Deprecated) The namespace that the Kafka Connect cluster is expected to reside in. This is only used in (*_app-interface_*) and (*_operator_*) modes."
   "``connectClusterName`` (string)", "(Deprecated) Defines the kafka connect cluster name that is used in this environment."
   "``suffix`` (string)", "(Deprecated) (Unused)"


.. _KafkaConnectClusterConfig :

KafkaConnectClusterConfig 
^^^^^^^^^^^^^^^^^^^^^^^^^

KafkaConnectClusterConfig defines options related to the Kafka Connect cluster managed/monitored by Clowder

Appears In:
:ref:`KafkaConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``name`` (string)", "Defines the kafka connect cluster name (default: '<kafka cluster's name>-connect')"
   "``namespace`` (string)", "The namespace the kafka connect cluster is expected to reside in (default: the kafka cluster's namespace)"
   "``replicas`` (integer)", "The requested number of replicas for kafka connect. If unset, default is '1'"
   "``version`` (string)", "Version. If unset, default is '2.5.0'"
   "``image`` (string)", "Image. If unset, default is 'quay.io/cloudservices/xjoin-kafka-connect-strimzi:latest'"


.. _KafkaTopicSpec :

KafkaTopicSpec 
^^^^^^^^^^^^^^

KafkaTopicSpec defines the desired state of KafkaTopic

Appears In:
:ref:`ClowdAppSpec`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``config`` (object (keys:string, values:string))", "A key/value pair describing the configuration of a particular topic."
   "``partitions`` (integer)", "The requested number of partitions for this topic. If unset, default is '3'"
   "``replicas`` (integer)", "The requested number of replicas for this topic. If unset, default is '3'"
   "``topicName`` (string)", "The requested name for this topic."


.. _LoggingConfig :

LoggingConfig 
^^^^^^^^^^^^^

LoggingConfig configures the Clowder provider controlling the creation of Logging instances.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``mode`` (LoggingMode)", "The mode of operation of the Clowder Logging Provider. Valid options are: (*_app-interface_*) where the provider will pass through cloudwatch credentials to the app configuration, and (*_none_*) where no logging will be configured."


.. _MetricsConfig :

MetricsConfig 
^^^^^^^^^^^^^

MetricsConfig configures the Clowder provider controlling the creation of metrics services and their probes.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``port`` (integer)", "The port that metrics services inside ClowdApp pods should be served on."
   "``path`` (string)", "A prefix path that pods will be instructed to use when setting up their metrics server."
   "``mode`` (MetricsMode)", "The mode of operation of the Metrics provider. The allowed modes are  (*_none_*), which disables metrics service generation, or (*_operator_*) where services and probes are generated. (*_app-interface_*) where services and probes are generated for app-interface."
   "``prometheus`` (:ref:`PrometheusConfig`)", "Prometheus specific configuration"






.. _ObjectStoreConfig :

ObjectStoreConfig 
^^^^^^^^^^^^^^^^^

ObjectStoreConfig configures the Clowder provider controlling the creation of ObjectStore instances.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``mode`` (ObjectStoreMode)", "The mode of operation of the Clowder ObjectStore Provider. Valid options are: (*_app-interface_*) where the provider will pass through Amazon S3 credentials to the app configuration, and (*_minio_*) where a local Minio instance will be created."
   "``suffix`` (string)", "Currently unused."
   "``pvc`` (boolean)", "If using the (*_local_*) mode and PVC is set to true, this instructs the local Database instance to use a PVC instead of emptyDir for its volumes."


.. _PodSpec :

PodSpec 
^^^^^^^

PodSpec defines a container running inside a ClowdApp.

Appears In:
:ref:`Deployment`
:ref:`Job`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``image`` (string)", "Image refers to the container image used to create the pod."
   "``initContainers`` (:ref:`InitContainer`)", "A list of init containers used to perform at-startup operations."
   "``command`` (string array)", "The command that will be invoked inside the pod at startup."
   "``args`` (string array)", "A list of args to be passed to the pod container."
   "``env`` (`EnvVar <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#envvar-v1-core>`_ array)", "A list of environment variables in k8s defined format."
   "``resources`` (`ResourceRequirements <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#resourcerequirements-v1-core>`_)", "A pass-through of a resource requirements in k8s ResourceRequirements format. If omitted, the default resource requirements from the ClowdEnvironment will be used."
   "``livenessProbe`` (`Probe <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#probe-v1-core>`_)", "A pass-through of a Liveness Probe specification in standard k8s format. If omitted, a standard probe will be setup point to the webPort defined in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to false."
   "``readinessProbe`` (`Probe <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#probe-v1-core>`_)", "A pass-through of a Readiness Probe specification in standard k8s format. If omitted, a standard probe will be setup point to the webPort defined in the ClowdEnvironment and a path of /healthz. Ignored if Web is set to false."
   "``volumes`` (`Volume <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#volume-v1-core>`_ array)", "A pass-through of a list of Volumes in standa k8s format."
   "``volumeMounts`` (`VolumeMount <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#volumemount-v1-core>`_ array)", "A pass-through of a list of VolumesMounts in standa k8s format."


.. _PodSpecDeprecated :

PodSpecDeprecated 
^^^^^^^^^^^^^^^^^

PodSpecDeprecated is a deprecated in favour of using the real k8s PodSpec object.

Appears In:
:ref:`ClowdAppSpec`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``name`` (string)", ""
   "``web`` (WebDeprecated)", ""
   "``minReplicas`` (integer)", ""
   "``image`` (string)", ""
   "``initContainers`` (:ref:`InitContainer`)", ""
   "``command`` (string array)", ""
   "``args`` (string array)", ""
   "``env`` (`EnvVar <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#envvar-v1-core>`_ array)", ""
   "``resources`` (`ResourceRequirements <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#resourcerequirements-v1-core>`_)", ""
   "``livenessProbe`` (`Probe <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#probe-v1-core>`_)", ""
   "``readinessProbe`` (`Probe <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#probe-v1-core>`_)", ""
   "``volumes`` (`Volume <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#volume-v1-core>`_ array)", ""
   "``volumeMounts`` (`VolumeMount <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#volumemount-v1-core>`_ array)", ""


.. _PrivateWebService :

PrivateWebService 
^^^^^^^^^^^^^^^^^

PrivateWebService is the definition of the private web service. There can be only one private service managed by Clowder.

Appears In:
:ref:`WebServices`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``enabled`` (boolean)", "Enabled describes if Clowder should enable the private service and provide the configuration in the cdappconfig."


.. _PrometheusConfig :

PrometheusConfig 
^^^^^^^^^^^^^^^^



Appears In:
:ref:`MetricsConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``deploy`` (boolean)", "Determines whether to deploy prometheus in operator mode"


.. _ProvidersConfig :

ProvidersConfig 
^^^^^^^^^^^^^^^

ProvidersConfig defines a group of providers configuration for a ClowdEnvironment.

Appears In:
:ref:`ClowdEnvironmentSpec`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``db`` (:ref:`DatabaseConfig`)", "Defines the Configuration for the Clowder Database Provider."
   "``inMemoryDb`` (:ref:`InMemoryDBConfig`)", "Defines the Configuration for the Clowder InMemoryDB Provider."
   "``kafka`` (:ref:`KafkaConfig`)", "Defines the Configuration for the Clowder Kafka Provider."
   "``logging`` (:ref:`LoggingConfig`)", "Defines the Configuration for the Clowder Logging Provider."
   "``metrics`` (:ref:`MetricsConfig`)", "Defines the Configuration for the Clowder Metrics Provider."
   "``objectStore`` (:ref:`ObjectStoreConfig`)", "Defines the Configuration for the Clowder ObjectStore Provider."
   "``web`` (:ref:`WebConfig`)", "Defines the Configuration for the Clowder Web Provider."
   "``featureFlags`` (:ref:`FeatureFlagsConfig`)", "Defines the Configuration for the Clowder FeatureFlags Provider."
   "``serviceMesh`` (:ref:`ServiceMeshConfig`)", "Defines the Configuration for the Clowder ServiceMesh Provider."
   "``pullSecrets`` (string array)", "Defines the pull secret to use for the service accounts."


.. _PublicWebService :

PublicWebService 
^^^^^^^^^^^^^^^^

PublicWebService is the definition of the public web service. There can be only one public service managed by Clowder.

Appears In:
:ref:`WebServices`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``enabled`` (boolean)", "Enabled describes if Clowder should enable the public service and provide the configuration in the cdappconfig."




.. _ServiceMeshConfig :

ServiceMeshConfig 
^^^^^^^^^^^^^^^^^

ServiceMeshConfig determines if this env should be part of a service mesh and, if enabled, configures the service mesh

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``mode`` (ServiceMeshMode)", ""


.. _WebConfig :

WebConfig 
^^^^^^^^^

WebConfig configures the Clowder provider controlling the creation of web services and their probes.

Appears In:
:ref:`ProvidersConfig`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``port`` (integer)", "The port that web services inside ClowdApp pods should be served on."
   "``privatePort`` (integer)", "The private port that web services inside a ClowdApp should be served on."
   "``apiPrefix`` (string)", "An api prefix path that pods will be instructed to use when setting up their web server."
   "``mode`` (WebMode)", "The mode of operation of the Web provider. The allowed modes are (*_none_*), which disables web service generation, or (*_operator_*) where services and probes are generated."


.. _WebServices :

WebServices 
^^^^^^^^^^^

WebServices defines the structs for the three exposed web services: public, private and metrics.

Appears In:
:ref:`Deployment`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``public`` (:ref:`PublicWebService`)", ""
   "``private`` (:ref:`PrivateWebService`)", ""
   "``metrics`` (:ref:`MetricsWebService`)", ""



.. _cyndi.cloud.redhat.com/v1alpha1:

cyndi.cloud.redhat.com/v1alpha1
-------------------------------

Package v1alpha1 contains API Schema definitions for the cyndi v1alpha1 API group

Resource Types
**************

- :ref:`CyndiPipeline`
- :ref:`CyndiPipelineList`



.. _CyndiPipeline :

CyndiPipeline 
^^^^^^^^^^^^^

CyndiPipeline is the Schema for the cyndipipelines API

Appears In:
:ref:`CyndiPipelineList`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cyndi.cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`CyndiPipeline`"
   "``metadata`` (`ObjectMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#objectmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``spec`` (:ref:`CyndiPipelineSpec`)", ""


.. _CyndiPipelineList :

CyndiPipelineList 
^^^^^^^^^^^^^^^^^

CyndiPipelineList contains a list of CyndiPipeline




.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``apiVersion`` (string)", "`cyndi.cloud.redhat.com/v1alpha1`"
      "``kind`` (string)", "`CyndiPipelineList`"
   "``metadata`` (`ListMeta <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#listmeta-v1-meta>`_)", "Refer to Kubernetes API documentation for fields of `metadata`."

   "``items`` (:ref:`CyndiPipeline`)", ""


.. _CyndiPipelineSpec :

CyndiPipelineSpec 
^^^^^^^^^^^^^^^^^

CyndiPipelineSpec defines the desired state of CyndiPipeline

Appears In:
:ref:`CyndiPipeline`


.. csv-table:: 
   :header: "Field", "Description"
   :widths: 10, 40

   "``appName`` (string)", ""
   "``insightsOnly`` (boolean)", ""
   "``connectCluster`` (string)", ""
   "``maxAge`` (integer)", ""
   "``validationThreshold`` (integer)", ""
   "``topic`` (string)", ""
   "``dbSecret`` (string)", ""
   "``inventoryDbSecret`` (string)", ""




