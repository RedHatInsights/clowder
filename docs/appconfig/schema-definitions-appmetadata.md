# AppMetadata Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata
```

Arbitrary metadata pertaining to the application application


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## AppMetadata Type

`object` ([AppMetadata](schema-definitions-appmetadata.md))

# AppMetadata Properties

| Property                    | Type     | Required | Nullable       | Defined by                                                                                                                                                                      |
| :-------------------------- | -------- | -------- | -------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [name](#name)               | `string` | Optional | cannot be null | [AppConfig](schema-definitions-appmetadata-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/name")               |
| [envName](#envname)         | `string` | Optional | cannot be null | [AppConfig](schema-definitions-appmetadata-properties-envname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/envName")         |
| [deployments](#deployments) | `array`  | Optional | cannot be null | [AppConfig](schema-definitions-appmetadata-properties-deployments.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/deployments") |

## name

Name of the ClowdApp


`name`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appmetadata-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/name")

### name Type

`string`

## envName

Name of the ClowdEnvironment this ClowdApp runs in


`envName`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appmetadata-properties-envname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/envName")

### envName Type

`string`

## deployments

Metadata pertaining to an application's deployments


`deployments`

-   is optional
-   Type: `object[]` ([DeploymentMetadata](schema-definitions-deploymentmetadata.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appmetadata-properties-deployments.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/deployments")

### deployments Type

`object[]` ([DeploymentMetadata](schema-definitions-deploymentmetadata.md))
