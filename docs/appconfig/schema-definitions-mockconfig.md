# MockConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/MockConfig
```

Mocked information


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## MockConfig Type

`object` ([MockConfig](schema-definitions-mockconfig.md))

# MockConfig Properties

| Property              | Type     | Required | Nullable       | Defined by                                                                                                                                                              |
| :-------------------- | -------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [bop](#bop)           | `string` | Optional | cannot be null | [AppConfig](schema-definitions-mockconfig-properties-bop.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/MockConfig/properties/bop")           |
| [keycloak](#keycloak) | `string` | Optional | cannot be null | [AppConfig](schema-definitions-mockconfig-properties-keycloak.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/MockConfig/properties/keycloak") |

## bop

BOP URL


`bop`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-mockconfig-properties-bop.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/MockConfig/properties/bop")

### bop Type

`string`

## keycloak

Keycloak


`keycloak`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-mockconfig-properties-keycloak.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/MockConfig/properties/keycloak")

### keycloak Type

`string`
