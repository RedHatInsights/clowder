# KeycloakConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig
```

Keycloak Config


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## KeycloakConfig Type

`object` ([KeycloakConfig](schema-definitions-keycloakconfig.md))

# KeycloakConfig Properties

| Property                            | Type     | Required | Nullable       | Defined by                                                                                                                                                                                    |
| :---------------------------------- | -------- | -------- | -------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [url](#url)                         | `string` | Optional | cannot be null | [AppConfig](schema-definitions-keycloakconfig-properties-url.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/url")                         |
| [username](#username)               | `string` | Optional | cannot be null | [AppConfig](schema-definitions-keycloakconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/username")               |
| [password](#password)               | `string` | Optional | cannot be null | [AppConfig](schema-definitions-keycloakconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/password")               |
| [defaultUsername](#defaultusername) | `string` | Optional | cannot be null | [AppConfig](schema-definitions-keycloakconfig-properties-defaultusername.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/defaultUsername") |
| [defaultPassword](#defaultpassword) | `string` | Optional | cannot be null | [AppConfig](schema-definitions-keycloakconfig-properties-defaultpassword.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/defaultPassword") |

## url

URL of Keycloak server


`url`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-keycloakconfig-properties-url.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/url")

### url Type

`string`

## username

URL of Keycloak server


`username`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-keycloakconfig-properties-username.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/username")

### username Type

`string`

## password

URL of Keycloak server


`password`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-keycloakconfig-properties-password.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/password")

### password Type

`string`

## defaultUsername

URL of Keycloak server


`defaultUsername`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-keycloakconfig-properties-defaultusername.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/defaultUsername")

### defaultUsername Type

`string`

## defaultPassword

URL of Keycloak server


`defaultPassword`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-keycloakconfig-properties-defaultpassword.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/KeycloakConfig/properties/defaultPassword")

### defaultPassword Type

`string`
