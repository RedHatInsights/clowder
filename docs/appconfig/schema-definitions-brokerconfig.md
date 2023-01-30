# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig
```

Broker Configuration


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## BrokerConfig Type

`object` ([Details](schema-definitions-brokerconfig.md))

# undefined Properties

| Property                              | Type      | Required | Nullable       | Defined by                                                                                                                                                                                  |
| :------------------------------------ | --------- | -------- | -------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [hostname](#hostname)                 | `string`  | Required | cannot be null | [AppConfig](schema-definitions-brokerconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/hostname")                 |
| [port](#port)                         | `integer` | Optional | cannot be null | [AppConfig](schema-definitions-brokerconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/port")                         |
| [cacert](#cacert)                     | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-brokerconfig-properties-cacert.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/cacert")                     |
| [authtype](#authtype)                 | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-brokerconfig-properties-authtype.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/authtype")                 |
| [sasl](#sasl)                         | `object`  | Optional | cannot be null | [AppConfig](schema-definitions-kafkasaslconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/sasl")                                      |
| [securityProtocol](#securityprotocol) | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-brokerconfig-properties-securityprotocol.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/securityProtocol") |

## hostname




`hostname`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-brokerconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/hostname")

### hostname Type

`string`

## port




`port`

-   is optional
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-brokerconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/port")

### port Type

`integer`

## cacert




`cacert`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-brokerconfig-properties-cacert.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/cacert")

### cacert Type

`string`

## authtype




`authtype`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-brokerconfig-properties-authtype.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/authtype")

### authtype Type

`string`

### authtype Constraints

**enum**: the value of this property must be equal to one of the following values:

| Value    | Explanation |
| :------- | ----------- |
| `"mtls"` |             |
| `"sasl"` |             |

## sasl

SASL Configuration for Kafka


`sasl`

-   is optional
-   Type: `object` ([Details](schema-definitions-kafkasaslconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-kafkasaslconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/sasl")

### sasl Type

`object` ([Details](schema-definitions-kafkasaslconfig.md))

## securityProtocol




`securityProtocol`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-brokerconfig-properties-securityprotocol.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/BrokerConfig/properties/securityProtocol")

### securityProtocol Type

`string`
