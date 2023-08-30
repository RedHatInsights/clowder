# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig
```

In Memory DB Configuration


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## InMemoryDBConfig Type

`object` ([Details](schema-definitions-inmemorydbconfig.md))

# undefined Properties

| Property              | Type      | Required | Nullable       | Defined by                                                                                                                                                                          |
| :-------------------- | --------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [hostname](#hostname) | `string`  | Required | cannot be null | [AppConfig](schema-definitions-inmemorydbconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/hostname") |
| [port](#port)         | `integer` | Required | cannot be null | [AppConfig](schema-definitions-inmemorydbconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/port")         |
| [username](#username) | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-inmemorydbconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/username") |
| [password](#password) | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-inmemorydbconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/password") |
| [sslMode](#sslmode)   | `boolean` | Optional | cannot be null | [AppConfig](schema-definitions-inmemorydbconfig-properties-sslmode.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/sslMode")   |

## hostname

Defines the hostname for the In Memory DB server configuration.


`hostname`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-inmemorydbconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/hostname")

### hostname Type

`string`

## port

Defines the port for the In Memory DB server configuration.


`port`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-inmemorydbconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/port")

### port Type

`integer`

## username

Defines the username for the In Memory DB server configuration.


`username`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-inmemorydbconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/username")

### username Type

`string`

## password

Defines the password for the In Memory DB server configuration.


`password`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-inmemorydbconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/password")

### password Type

`string`

## sslMode

Defines the sslMode used by the In Memory DB server coniguration


`sslMode`

-   is optional
-   Type: `boolean`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-inmemorydbconfig-properties-sslmode.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/InMemoryDBConfig/properties/sslMode")

### sslMode Type

`boolean`
