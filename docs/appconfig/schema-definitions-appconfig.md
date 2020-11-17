# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig
```

ClowdApp deployment configuration for Clowder enabled apps.


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                          |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../../../out/schema.json "open original schema") |

## AppConfig Type

`object` ([Details](schema-definitions-appconfig.md))

# undefined Properties

| Property                    | Type      | Required | Nullable       | Defined by                                                                                                                                                                  |
| :-------------------------- | --------- | -------- | -------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [webPort](#webport)         | `integer` | Required | cannot be null | [AppConfig](schema-definitions-appconfig-properties-webport.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/webPort")         |
| [metricsPort](#metricsport) | `integer` | Required | cannot be null | [AppConfig](schema-definitions-appconfig-properties-metricsport.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/metricsPort") |
| [metricsPath](#metricspath) | `string`  | Required | cannot be null | [AppConfig](schema-definitions-appconfig-properties-metricspath.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/metricsPath") |
| [logging](#logging)         | `object`  | Required | cannot be null | [AppConfig](schema-definitions-loggingconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/logging")                        |
| [kafka](#kafka)             | `object`  | Optional | cannot be null | [AppConfig](schema-definitions-kafkaconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/kafka")                            |
| [database](#database)       | `object`  | Optional | cannot be null | [AppConfig](schema-definitions-databaseconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/database")                      |
| [objectStore](#objectstore) | `object`  | Optional | cannot be null | [AppConfig](schema-definitions-objectstoreconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/objectStore")                |
| [inMemoryDb](#inmemorydb)   | `object`  | Optional | cannot be null | [AppConfig](schema-definitions-inmemorydbconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/inMemoryDb")                  |
| [endpoints](#endpoints)     | `array`   | Optional | cannot be null | [AppConfig](schema-definitions-appconfig-properties-endpoints.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/endpoints")     |

## webPort

Defines the web port that the app should be configured to listen on for API traffic.


`webPort`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appconfig-properties-webport.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/webPort")

### webPort Type

`integer`

## metricsPort

Defines the metrics port that the app should be configured to listen on for metric traffic.


`metricsPort`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appconfig-properties-metricsport.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/metricsPort")

### metricsPort Type

`integer`

## metricsPath

Defines the path to the metrics server that the app should be configured to listen on for metric traffic.


`metricsPath`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appconfig-properties-metricspath.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/metricsPath")

### metricsPath Type

`string`

## logging

Logging Configuration


`logging`

-   is required
-   Type: `object` ([LoggingConfig](schema-definitions-loggingconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-loggingconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/logging")

### logging Type

`object` ([LoggingConfig](schema-definitions-loggingconfig.md))

## kafka

Kafka Configuration


`kafka`

-   is optional
-   Type: `object` ([Details](schema-definitions-kafkaconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-kafkaconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/kafka")

### kafka Type

`object` ([Details](schema-definitions-kafkaconfig.md))

## database

Database Configuration


`database`

-   is optional
-   Type: `object` ([DatabaseConfig](schema-definitions-databaseconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/database")

### database Type

`object` ([DatabaseConfig](schema-definitions-databaseconfig.md))

## objectStore

Object Storage Configuration


`objectStore`

-   is optional
-   Type: `object` ([Details](schema-definitions-objectstoreconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstoreconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/objectStore")

### objectStore Type

`object` ([Details](schema-definitions-objectstoreconfig.md))

## inMemoryDb

In Memory DB Configuration


`inMemoryDb`

-   is optional
-   Type: `object` ([Details](schema-definitions-inmemorydbconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-inmemorydbconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/inMemoryDb")

### inMemoryDb Type

`object` ([Details](schema-definitions-inmemorydbconfig.md))

## endpoints




`endpoints`

-   is optional
-   Type: `object[]` ([Details](schema-definitions-dependencyendpoint.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appconfig-properties-endpoints.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppConfig/properties/endpoints")

### endpoints Type

`object[]` ([Details](schema-definitions-dependencyendpoint.md))
