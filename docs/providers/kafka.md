# Kafka Provider

The **Kafka Provider** is responsible for providing access to one or more Kafka
topics.

## ClowdApp Configuration

To request a Kafka topic, a `ClowdApp` would use the `kafkaTopics` stanza, a
partial example of which is shown below.

```yaml
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
```

## ClowdEnv Configuration

The **Kafka Provider** will run in one of the following modes. These are set up
by the ClowdEnvironment. Depending on the environment you are running you may
or may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

### managed

In managed mode, which is the current default for stage and production
environments the **Kafka Provider** will provide connection information for
Kafka enabled `ClowdApps` via cdappconfig. As in **app-interface** mode,
topic information from the `ClowdApp` will be passed through to the client
config. The topic resources themselves must be created via the [Platform-MQ](https://github.com/RedHatInsights/platform-mq/tree/main/helm#readme)
resource repository.

### operator

In operator mode, the **Kafka Provider** will provision KafkaTopic CRs to be
consumed by the Strimzi operator. If multiple apps request the same topic, a
mathematical `max` operation will be performed on any partitions, replicas as
well as numerical configuration field, before applying these to the KafkaTopic
CR. The CR will be placed in the environment specified in the `ClowdEnv`. The
topic name will be modified as described above, to facilitate using the same
Kafka instance for multiple apps in differing environments.

ClowdEnv Config options available:

- `clusterName`
- `namespace`
- `connectNamespace`
- `connectClusterName`

### app-interface

In app-interface mode, the Clowder operator does not create any resources and
simply passes through the topic names from the `ClowdApp` to the client
config. The topics should be created via the [Platform-MQ](https://github.com/RedHatInsights/platform-mq/tree/main/helm#readme) resource repository.

ClowdEnv Config options available:

- `clusterName`
- `namespace`
- `connectNamespace`
- `connectClusterName`

## Generated App Configuration

The Kafka configuration appears in the cdappconfig.json with the following
structure. The name that was requested in the `ClowdApp` will be presented as
the `requestedName` attribute in the topic object. The kafka provider modifies
the name in some modes where a single kafka instance is shared between multiple
environments. This allows the same topic name to be requested by apps
in different environments without them polluting each other. Apps should use
the `name` attribute of a topic when connecting to Kafka.

A helper is available below to facilitate quick access via a map.

If a the `cacert` and `securityProtocol` fields are populated, then the kafka
instance will be using TLS and the client should be configured as such

NOTE: The `securityProtocol` field has been deprecated from within the SASL
stanza and in the future will only be available at the top level broker
stanza.

### JSON structure

```json
{
  "kafka": {
      "brokers": [
          {
              "hostname": "broker-host",
              "port": 27015,
              "securityProtocol": "SSL",
              "cacert": "<some CA string>"
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
```

A User auth enabled Kafka will look like this

```json
{
  "kafka": {
      "brokers": [
          {
              "hostname": "broker-host",
              "port": 27015,
              "authtype": "sasl",
              "cacert": "-----BEGIN CERTIFICATE-----\nMIIDLTCCAhWgAwIBAgIJAPOWU.........",
              "sasl":{
                  "username": "kafkausername",
                  "password": "kafkapassword",
                  "saslMechanism": "SCRAM-SHA-512"
                  "securityProtocol": "SASL_SSL"
              },
              "securityProtocol": "SASL_SSL"
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
```


### Client access

For supported languages, the kafka configuration is accessed via the following
attribute names.

| Language   | Attribute Name        |
|------------|-----------------------|
| Python     | `LoadedConfig.kafka`  |
| Go         | `LoadedConfig.Kafka`  |
| JavaScript | `LoadedConfig.kafka`  |
| Ruby       | `LoadedConfig.kafka`  |


### Client helpers

`KafkaTopics` Returns a map of topic objects, using the original requested name
as the key and the topic object the value. `KafkaServers` Returns a list of
Kafka broker strings comprising of hostname and port.

| Name        | Kafka Topics                                                                 | Kafka Servers                                                           |
|-------------|------------------------------------------------------------------------------|-------------------------------------------------------------------------|
| Description | Returns a map of topic objects, using the original requested name as the key and the topic object as the value. | Returns a list of Kafka broker strings comprising of hostname and port. |
| Python      | `KafkaTopics`                                                                | `KafkaServers`                                                          |
| Go          | `KafkaTopics`                                                                | `KafkaServers`                                                          |
| JavaScript  | `KafkaTopics`                                                                | `KafkaServers`                                                          |
| Ruby        | `KafkaTopics`                                                                | `KafkaServers`                                                          |


### ClowdEnv Configuration

Configuring the **Kafka Provider** is done by providing the follow JSON structure
to the ``ClowdEnv`` resource. Further details of the options available can be
found in the API reference. A minimal example is shown below for the
``operator`` mode. Different modes can use different configuration options,
more information can be found in the API reference.

```yaml
    apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdEnvironment
    metadata:
      name: myenv
    spec:
      # Other Env Config
      providers:
        kafka:
          mode: operator
          pvc: false
```


## Cyndi

The *Cyndi* attribute of a ClowdApp definition is responsible for ensuring the Cyndi host
syndication process is in place for the ClowdApp. On Clowder managed environments, the
provider is also responsible of creating the CyndiPipeline resource, and configuring the underlying
`Kafka Connect` so that the data syndication from Host Inventory database to the app's database
works correctly.

[Kafka Connect](https://docs.confluent.io/platform/current/connect/index.html#kafka-connect) is the core component used to perform Inventory’s host database
syndication for some of the Insights platform applications, and uses an Operator to orchestrate
the synchronization process.

It does so by using a `CyndiPipeline` resource, which can be created by Clowder, and by injecting
in the database secrets for both the target db (the application’s) and the host inventory db.

[Kafka Connect](https://docs.confluent.io/platform/current/connect/index.html#kafka-connect) streams table updates to keep data syndicated between the host
inventory db and the application’s hosts view.

Please refer to the corresponing projects for more information about the
[Cyndi project](https://consoledot.pages.redhat.com/docs/dev/services/inventory.html#cyndi), the `CyndiPipeline` resource or the [Cyndi Operator](https://github.com/RedHatInsights/cyndi-operator#cyndi-operator)
itself.

### ClowdApp Configuration

In order to request a `ClowdApp` to get the host syndication enabled by Cyndi, the `Cyndi` stanza
needs to be used. A snippet of how that config would look like follows.

```yaml
# my-clowdapp.yml
# ...
cyndi:
  enabled: true
  appName: fancyapp
  insightsOnly: true
# ...
```

The attributes are described in Clowder’s API Spec documentation [here](https://redhatinsights.github.io/clowder/clowder/dev/api_reference)

* *enabled* `[bool] default: true` - enables or disables the Cyndi dependency for this particular ClowdApp resource.
* *appName* `[str] default: ''` - a string that sets the unique identifier of this ClowdApp on Cyndi.
* *insightsOnly* `[bool] default: false` - enables the data syndication for all hosts or Insights hosts only.

### ClowdEnv Configuration

The *Cyndi* provider will run in one of the two following modes, depending on whether if Clowder
manages the environment or not;

On non-Clowder managed environments (at the time of this writing, Stage and Production) Clowder
will only check that the CyndiPipeline resource is available for the Clowdapp that is being
reconciled in case it has the “cyndi” flag enabled in it’s template definition.
Clowder will not try to create or update any resource related to Cyndi on environments it does not
manage.

On Clowder managed environments (Ephemeral environment) Clowder will configure and deploy Kafka
using the Strimzi operator and will setup the CyndiPipeline to enable the host syndication process
for the Clowdapps that require it on their spec files (see [Clowder API reference](https://redhatinsights.github.io/clowder/clowder/dev/api_reference))

