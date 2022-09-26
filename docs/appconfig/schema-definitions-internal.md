# Internal Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/Internal
```




| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## Internal Type

`object` ([Internal](schema-definitions-internal.md))

# Internal Properties

| Property              | Type     | Required | Nullable       | Defined by                                                                                                                                            |
| :-------------------- | -------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------- |
| [keycloak](#keycloak) | `object` | Optional | cannot be null | [AppConfig](schema-definitions-keycloakconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/Internal/properties/keycloak") |
| [database](#database) | `array`  | Optional | cannot be null | [AppConfig](schema-definitions-shareddatabase.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/Internal/properties/database") |

## keycloak

Keycloak Config


`keycloak`

-   is optional
-   Type: `object` ([KeycloakConfig](schema-definitions-keycloakconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-keycloakconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/Internal/properties/keycloak")

### keycloak Type

`object` ([KeycloakConfig](schema-definitions-keycloakconfig.md))

## database

Shared Database Config


`database`

-   is optional
-   Type: `object[]` ([SharedDatabaseConfig](schema-definitions-shareddatabaseconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-shareddatabase.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/Internal/properties/database")

### database Type

`object[]` ([SharedDatabaseConfig](schema-definitions-shareddatabaseconfig.md))
