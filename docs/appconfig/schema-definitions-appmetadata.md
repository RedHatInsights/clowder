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

| Property                    | Type    | Required | Nullable       | Defined by                                                                                                                                                                      |
| :-------------------------- | ------- | -------- | -------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [deployments](#deployments) | `array` | Optional | cannot be null | [AppConfig](schema-definitions-appmetadata-properties-deployments.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/deployments") |

## deployments

Metadata pertaining to an application's deployments


`deployments`

-   is optional
-   Type: `object[]` ([DeploymentMetadata](schema-definitions-deploymentmetadata.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-appmetadata-properties-deployments.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/AppMetadata/properties/deployments")

### deployments Type

`object[]` ([DeploymentMetadata](schema-definitions-deploymentmetadata.md))
