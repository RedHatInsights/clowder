# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig
```

Feature Flags Configuration


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## FeatureFlagsConfig Type

`object` ([Details](schema-definitions-featureflagsconfig.md))

# undefined Properties

| Property                                | Type      | Required | Nullable       | Defined by                                                                                                                                                                                                |
| :-------------------------------------- | --------- | -------- | -------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [hostname](#hostname)                   | `string`  | Required | cannot be null | [AppConfig](schema-definitions-featureflagsconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/hostname")                   |
| [port](#port)                           | `integer` | Required | cannot be null | [AppConfig](schema-definitions-featureflagsconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/port")                           |
| [clientAccessToken](#clientaccesstoken) | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-featureflagsconfig-properties-clientaccesstoken.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/clientAccessToken") |
| [scheme](#scheme)                       | `string`  | Required | cannot be null | [AppConfig](schema-definitions-featureflagsconfig-properties-scheme.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/scheme")                       |

## hostname

Defines the hostname for the FeatureFlags server


`hostname`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-featureflagsconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/hostname")

### hostname Type

`string`

## port

Defines the port for the FeatureFlags server


`port`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-featureflagsconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/port")

### port Type

`integer`

## clientAccessToken

Defines the client access token to use when connect to the FeatureFlags server


`clientAccessToken`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-featureflagsconfig-properties-clientaccesstoken.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/clientAccessToken")

### clientAccessToken Type

`string`

## scheme

Details the scheme to use for FeatureFlags http/https


`scheme`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-featureflagsconfig-properties-scheme.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/FeatureFlagsConfig/properties/scheme")

### scheme Type

`string`

### scheme Constraints

**enum**: the value of this property must be equal to one of the following values:

| Value     | Explanation |
| :-------- | ----------- |
| `"http"`  |             |
| `"https"` |             |
