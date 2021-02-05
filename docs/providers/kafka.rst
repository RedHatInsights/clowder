..  _kafkaprovider:

Kafka Provider
==============

The **Kafka Provider** is responsible for providing access to one or more Kafka
topics.

ClowdApp Configuration
----------------------

To request a Kafka topic, a ``ClowdApp`` would use the `kafkaTopics` stanza, a
partial example of which is shown below.

.. code-block:: yaml

    apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdApp
    metadata:
      name: myapp
    spec:
      # Other App Config
      kafkaTopics:
      - replicas: 5
        partitions: 5
        topicName: topicOne
      - replicas: 5
        partitions: 5
        topicName: topicTwo
        config:
          retention.ms: "234234234"
          retention.bytes: "2352352"

ClowdEnv Configuration
----------------------

Modes
*****

The **Kafka Provider** will run in one of the following modes. These are set up by
the ClowdEnvironment. Depending on the environment you are running you may or
may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

local
^^^^^

In local mode, the **Kafka Provider** will provision a single node kafka instance
in the namespace defined in the ``ClowdEnv`` for the environment. This will
comprise of a Zookeeper and Kafka pod. Topics will be created with minimal
replicas/partitions as it is only a single node instance.

ClowdEnv Config options available:

- ``pvc``

operator
^^^^^^^^

In operator mode, the **Kafka Provider** will provision KafkaTopic CRs to be
consumed by the Strimzi operator. If multiple apps request the same topic, a
mathematical ``max`` operation will be performed on any partitions, replicas as
well as numerical configuration field, before applying these to the KafkaTopic
CR. The CR will be placed in the environment specified in the ``ClowdEnv``. The
topic name will be modified as described above, to facilitate using the same
Kafka instance for multiple apps in differing environments.

ClowdEnv Config options available:

- ``clusterName``
- ``namespace``
- ``connectNamespace``
- ``connectClusterName``

app-interface
^^^^^^^^^^^^^

In app-interface mode, the Clowder operator does not create any resources and
simply passes through the topic names from the ``ClowdApp`` to the client
config. The topics should be created via the usual app-interface means.

ClowdEnv Config options available:

- ``clusterName``
- ``namespace``
- ``connectNamespace``
- ``connectClusterName``

Generated App Configuration
---------------------------

The Kafka configuration appears in the cdappconfig.json with the following
structure. The name that was requested in the ``ClowdApp`` will be presented as
the ``requestedName`` attribute in the topic object. The kafka provider modifies
the name in some modes where a single kafka instance is shared between multiple
environments. This allows the same topic name to be requested by apps
in different environments without them polluting each other. Apps should use
the `name` attribute of a topic when connecting to Kafka.

A helper is available below to facilitate quick access via a map.

JSON structure
**************

.. code-block:: json

    {
      "kafka": {
          "brokers": [
              {
                  "hostname": "broker-host",
                  "port": 27015
              }
          ],
          "topics": [
              {
                  "requestedName": "originalName",
                  "name": "someTopic",
                  "consumerGroupName": "someGroupName"
              }
          ]
      }
    }

Client access
*************

For supported languages, the kafka configuration is access via the following
attribute names.

+-----------+------------------------+
| Language  | Attribute Name         |
+===========+========================+
| Python    | ``LoadedConfig.kafka`` |
+-----------+------------------------+
| Go        | ``LoadedConfig.Kafka`` |
+-----------+------------------------+
| Javscript | ``LoadedConfig.kafka`` |
+-----------+------------------------+
| Ruby      | ``LoadedConfig.kafka`` |
+-----------+------------------------+


Client helpers
**************

+-------------+-----------------------------------+--------------------------------+
| Name        | Kafka Topics                      | Kafka Servers                  |
+=============+===================================+================================+
| Description | Returns a map of topic objects,   | Returns a list of Kafka broker |
|             |                                   |                                |
|             | using the original requested name | strings comprising of hostname |
|             |                                   |                                |
|             | as the key and the topic object   | and port.                      |
|             |                                   |                                | 
|             | as the value.                     |                                |
+-------------+-----------------------------------+--------------------------------+
| Python      | ``KafkaTopics``                   | ``KafkaServers``               |
+-------------+-----------------------------------+--------------------------------+
| Go          | ``KafkaTopics``                   | ``KafkaServers``               |
+-------------+-----------------------------------+--------------------------------+
| Javscript   | ``KafkaTopics``                   | ``KafkaServers``               |
+-------------+-----------------------------------+--------------------------------+
| Ruby        | ``KafkaTopics``                   | ``KafkaServers``               |
+-------------+-----------------------------------+--------------------------------+

ClowdEnv Configuration
**********************

Configuring the **Kafka Provider** is done by providing the follow JSON structure
to the ``ClowdEnv`` resource. Further details of the options available can be
found in the API reference. A minimal example is shown below for the
``operator`` mode. Different modes can use different configuration options,
more information can be found in the API reference.

.. code-block:: yaml

    apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdEnvivonment
    metadata:
      name: myenv
    spec:
      # Other Env Config
      providers:
        kafka:
          mode: local
          pvc: false
