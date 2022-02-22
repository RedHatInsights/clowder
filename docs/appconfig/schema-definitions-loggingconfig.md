# LoggingConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/LoggingConfig
```

Logging Configuration

| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                   |
| :------------------ | :--------- | :------------- | :----------- | :---------------- | :-------------------- | :------------------ | :----------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json*](../../out/schema.json "open original schema") |

## LoggingConfig Type

`object` ([LoggingConfig](schema-definitions-loggingconfig.md))

# LoggingConfig Properties

| Property                  | Type     | Required | Nullable       | Defined by                                                                                                                                                       |
| :------------------------ | :------- | :------- | :------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [type](#type)             | `string` | Required | cannot be null | [AppConfig](schema-definitions-loggingconfig-properties-type.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/LoggingConfig/properties/type") |
| [cloudwatch](#cloudwatch) | `object` | Optional | cannot be null | [AppConfig](schema-definitions-cloudwatchconfig.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/LoggingConfig/properties/cloudwatch")        |

## type

Defines the type of logging configuration

`type`

*   is required

*   Type: `string`

*   cannot be null

*   defined in: [AppConfig](schema-definitions-loggingconfig-properties-type.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/LoggingConfig/properties/type")

### type Type

`string`

## cloudwatch

Cloud Watch configuration

`cloudwatch`

*   is optional

*   Type: `object` ([CloudWatchConfig](schema-definitions-cloudwatchconfig.md))

*   cannot be null

*   defined in: [AppConfig](schema-definitions-cloudwatchconfig.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/LoggingConfig/properties/cloudwatch")

### cloudwatch Type

`object` ([CloudWatchConfig](schema-definitions-cloudwatchconfig.md))
