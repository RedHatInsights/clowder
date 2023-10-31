# Untitled string in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/KafkaSASLConfig/properties/securityProtocol
```

Broker security protocol. DEPRECATED, use the top level securityProtocol field instead


| Abstract            | Extensible | Status         | Identifiable            | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ----------------------- | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | Unknown identifiability | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## securityProtocol Type

`string`

## securityProtocol Constraints

**enum**: the value of this property must be equal to one of the following values:

| Value        | Explanation |
| :----------- | ----------- |
| `"SASL_SSL"` |             |
| `"SSL"`      |             |
