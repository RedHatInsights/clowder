# TestingConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/TestingConfig
```

Config for Testing Spec in Job Settings


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## TestingConfig Type

`object` ([TestingConfig](schema-definitions-testingconfig.md))

# TestingConfig Properties

| Property                          | Type     | Required | Nullable       | Defined by                                                                                                                                                                                |
| :-------------------------------- | -------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [k8sAccessLevel](#k8saccesslevel) | `string` | Required | cannot be null | [AppConfig](schema-definitions-testingconfig-properties-k8saccesslevel.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/TestingConfig/properties/k8sAccessLevel") |
| [configAccess](#configaccess)     | `string` | Required | cannot be null | [AppConfig](schema-definitions-testingconfig-properties-configaccess.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/TestingConfig/properties/configAccess")     |
| [iqe](#iqe)                       | `object` | Optional | cannot be null | [AppConfig](schema-definitions-iqeconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/TestingConfig/properties/iqe")                                          |

## k8sAccessLevel

Defines the level of access the iqe pod has in the namespace


`k8sAccessLevel`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-testingconfig-properties-k8saccesslevel.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/TestingConfig/properties/k8sAccessLevel")

### k8sAccessLevel Type

`string`

## configAccess

Defines the amount of app config is mounted to the pod


`configAccess`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-testingconfig-properties-configaccess.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/TestingConfig/properties/configAccess")

### configAccess Type

`string`

## iqe

Config for IqeJob Settings


`iqe`

-   is optional
-   Type: `object` ([IqeConfig](schema-definitions-iqeconfig.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-iqeconfig.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/TestingConfig/properties/iqe")

### iqe Type

`object` ([IqeConfig](schema-definitions-iqeconfig.md))
