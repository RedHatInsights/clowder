# IqeConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/IqeConfig
```

Config for IqeJob Settings


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## IqeConfig Type

`object` ([IqeConfig](schema-definitions-iqeconfig.md))

# IqeConfig Properties

| Property                | Type     | Required | Nullable       | Defined by                                                                                                                                                              |
| :---------------------- | -------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [imageBase](#imagebase) | `string` | Required | cannot be null | [AppConfig](schema-definitions-iqeconfig-properties-imagebase.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/IqeConfig/properties/imageBase") |

## imageBase

Defines the base image used for iqe testing


`imageBase`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-iqeconfig-properties-imagebase.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/IqeConfig/properties/imageBase")

### imageBase Type

`string`
