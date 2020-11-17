# DatabaseConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig
```

Database Configuration


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                          |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../../../out/schema.json "open original schema") |

## DatabaseConfig Type

`object` ([DatabaseConfig](schema-definitions-databaseconfig.md))

# DatabaseConfig Properties

| Property                        | Type      | Required | Nullable       | Defined by                                                                                                                                                                                |
| :------------------------------ | --------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [name](#name)                   | `string`  | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/name")                   |
| [username](#username)           | `string`  | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/username")           |
| [password](#password)           | `string`  | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/password")           |
| [hostname](#hostname)           | `string`  | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/hostname")           |
| [port](#port)                   | `integer` | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/port")                   |
| [adminUsername](#adminusername) | `string`  | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-adminusername.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/adminUsername") |
| [adminPassword](#adminpassword) | `string`  | Required | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-adminpassword.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/adminPassword") |
| [rdsCa](#rdsca)                 | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-databaseconfig-properties-rdsca.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/rdsCa")                 |

## name

Defines the database name.


`name`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/name")

### name Type

`string`

## username

Defines a username with standard access to the database.


`username`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/username")

### username Type

`string`

## password

Defines the password for the standard user.


`password`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/password")

### password Type

`string`

## hostname

Defines the hostname of the database configured for the ClowdApp.


`hostname`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/hostname")

### hostname Type

`string`

## port

Defines the port of the database configured for the ClowdApp.


`port`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/port")

### port Type

`integer`

## adminUsername

Defines the pgAdmin username.


`adminUsername`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-adminusername.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/adminUsername")

### adminUsername Type

`string`

## adminPassword

Defines the pgAdmin password.


`adminPassword`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-adminpassword.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/adminPassword")

### adminPassword Type

`string`

## rdsCa

Defines the CA used to access the database.


`rdsCa`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-databaseconfig-properties-rdsca.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DatabaseConfig/properties/rdsCa")

### rdsCa Type

`string`
