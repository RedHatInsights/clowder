# DeploymentMetadata Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/DeploymentMetadata
```

Deployment Metadata

| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                   |
| :------------------ | :--------- | :------------- | :----------- | :---------------- | :-------------------- | :------------------ | :----------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json*](../../out/schema.json "open original schema") |

## DeploymentMetadata Type

`object` ([DeploymentMetadata](schema-definitions-deploymentmetadata.md))

# DeploymentMetadata Properties

| Property        | Type     | Required | Nullable       | Defined by                                                                                                                                                                   |
| :-------------- | :------- | :------- | :------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [name](#name)   | `string` | Required | cannot be null | [AppConfig](schema-definitions-deploymentmetadata-properties-name.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/DeploymentMetadata/properties/name")   |
| [image](#image) | `string` | Required | cannot be null | [AppConfig](schema-definitions-deploymentmetadata-properties-image.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/DeploymentMetadata/properties/image") |

## name

Name of deployment

`name`

*   is required

*   Type: `string`

*   cannot be null

*   defined in: [AppConfig](schema-definitions-deploymentmetadata-properties-name.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/DeploymentMetadata/properties/name")

### name Type

`string`

## image

Image used by deployment

`image`

*   is required

*   Type: `string`

*   cannot be null

*   defined in: [AppConfig](schema-definitions-deploymentmetadata-properties-image.md "https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/DeploymentMetadata/properties/image")

### image Type

`string`
