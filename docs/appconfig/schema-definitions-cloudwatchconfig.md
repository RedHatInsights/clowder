# CloudWatchConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig
```

Cloud Watch configuration


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## CloudWatchConfig Type

`object` ([CloudWatchConfig](schema-definitions-cloudwatchconfig.md))

# CloudWatchConfig Properties

| Property                            | Type     | Required | Nullable       | Defined by                                                                                                                                                                                        |
| :---------------------------------- | -------- | -------- | -------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [accessKeyId](#accesskeyid)         | `string` | Required | cannot be null | [AppConfig](schema-definitions-cloudwatchconfig-properties-accesskeyid.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/accessKeyId")         |
| [secretAccessKey](#secretaccesskey) | `string` | Required | cannot be null | [AppConfig](schema-definitions-cloudwatchconfig-properties-secretaccesskey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/secretAccessKey") |
| [region](#region)                   | `string` | Required | cannot be null | [AppConfig](schema-definitions-cloudwatchconfig-properties-region.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/region")                   |
| [logGroup](#loggroup)               | `string` | Required | cannot be null | [AppConfig](schema-definitions-cloudwatchconfig-properties-loggroup.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/logGroup")               |

## accessKeyId

Defines the access key that the app should use for configuring CloudWatch.


`accessKeyId`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-cloudwatchconfig-properties-accesskeyid.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/accessKeyId")

### accessKeyId Type

`string`

## secretAccessKey

Defines the secret key that the app should use for configuring CloudWatch.


`secretAccessKey`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-cloudwatchconfig-properties-secretaccesskey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/secretAccessKey")

### secretAccessKey Type

`string`

## region

Defines the region that the app should use for configuring CloudWatch.


`region`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-cloudwatchconfig-properties-region.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/region")

### region Type

`string`

## logGroup

Defines the logGroup that the app should use for configuring CloudWatch.


`logGroup`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-cloudwatchconfig-properties-loggroup.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/CloudWatchConfig/properties/logGroup")

### logGroup Type

`string`
