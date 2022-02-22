# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/sasl
```

SASL Configuration for Kafka


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## sasl Type

`object` ([Details](schema-definitions-kafkasaslconfig.md))

# undefined Properties

| Property              | Type     | Required | Nullable       | Defined by                                                                                                                                                                        |
| :-------------------- | -------- | -------- | -------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [username](#username) | `string` | Optional | cannot be null | [AppConfig](schema-definitions-kafkasaslconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaSASLConfig/properties/username") |
| [password](#password) | `string` | Optional | cannot be null | [AppConfig](schema-definitions-kafkasaslconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaSASLConfig/properties/password") |

## username




`username`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-kafkasaslconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaSASLConfig/properties/username")

### username Type

`string`

## password




`password`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-kafkasaslconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaSASLConfig/properties/password")

### password Type

`string`
