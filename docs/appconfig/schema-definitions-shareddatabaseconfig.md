# SharedDatabaseConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/SharedDatabaseConfig
```

Keycloak Config


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## SharedDatabaseConfig Type

`object` ([SharedDatabaseConfig](schema-definitions-shareddatabaseconfig.md))

# SharedDatabaseConfig Properties

| Property            | Type      | Required | Nullable       | Defined by                                                                                                                                                                                |
| :------------------ | --------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [config](#config)   | `object`  | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/SharedDatabaseConfig/properties/config")                           |
| [version](#version) | `integer` | Required | cannot be null | [AppConfig](schema-definitions-shareddatabaseconfig-properties-version.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/SharedDatabaseConfig/properties/version") |

## config

Database Configuration


`config`

-   is required
-   Type: `object` ([DatabaseConfig](schema-definitions-databaseconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/SharedDatabaseConfig/properties/config")

### config Type

`object` ([DatabaseConfig](schema-definitions-databaseconfig.md))

## version

Version number


`version`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-shareddatabaseconfig-properties-version.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/SharedDatabaseConfig/properties/version")

### version Type

`integer`
