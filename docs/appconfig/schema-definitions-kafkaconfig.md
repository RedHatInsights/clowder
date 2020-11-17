# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaConfig
```

Kafka Configuration


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                          |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../../../out/schema.json "open original schema") |

## KafkaConfig Type

`object` ([Details](schema-definitions-kafkaconfig.md))

# undefined Properties

| Property            | Type    | Required | Nullable       | Defined by                                                                                                                                                              |
| :------------------ | ------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [brokers](#brokers) | `array` | Required | cannot be null | [AppConfig](schema-definitions-kafkaconfig-properties-brokers.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaConfig/properties/brokers") |
| [topics](#topics)   | `array` | Required | cannot be null | [AppConfig](schema-definitions-kafkaconfig-properties-topics.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaConfig/properties/topics")   |

## brokers

Defines the brokers the app should connect to for Kafka services.


`brokers`

-   is required
-   Type: `object[]` ([Details](schema-definitions-brokerconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-kafkaconfig-properties-brokers.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaConfig/properties/brokers")

### brokers Type

`object[]` ([Details](schema-definitions-brokerconfig.md))

## topics

Defines a list of the topic configurations available to the application.


`topics`

-   is required
-   Type: `object[]` ([Details](schema-definitions-topicconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-kafkaconfig-properties-topics.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaConfig/properties/topics")

### topics Type

`object[]` ([Details](schema-definitions-topicconfig.md))
