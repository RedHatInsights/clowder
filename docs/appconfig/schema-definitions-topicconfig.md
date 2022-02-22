# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/TopicConfig
```

Topic Configuration

| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                   |
| :------------------ | :--------- | :------------- | :----------- | :---------------- | :-------------------- | :------------------ | :----------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json*](../../out/schema.json "open original schema") |

## TopicConfig Type

`object` ([Details](schema-definitions-topicconfig.md))

# TopicConfig Properties

| Property                        | Type     | Required | Nullable       | Defined by                                                                                                                                                                     |
| :------------------------------ | :------- | :------- | :------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [requestedName](#requestedname) | `string` | Required | cannot be null | [AppConfig](schema-definitions-topicconfig-properties-requestedname.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/TopicConfig/properties/requestedName") |
| [name](#name)                   | `string` | Required | cannot be null | [AppConfig](schema-definitions-topicconfig-properties-name.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/TopicConfig/properties/name")                   |

## requestedName

The name that the app requested in the ClowdApp definition.

`requestedName`

*   is required

*   Type: `string`

*   cannot be null

*   defined in: [AppConfig](schema-definitions-topicconfig-properties-requestedname.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/TopicConfig/properties/requestedName")

### requestedName Type

`string`

## name

The name of the actual topic on the Kafka server.

`name`

*   is required

*   Type: `string`

*   cannot be null

*   defined in: [AppConfig](schema-definitions-topicconfig-properties-name.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/TopicConfig/properties/name")

### name Type

`string`
