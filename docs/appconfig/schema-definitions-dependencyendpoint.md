# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint
```

Dependent service connection info


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## DependencyEndpoint Type

`object` ([Details](schema-definitions-dependencyendpoint.md))

# undefined Properties

| Property              | Type      | Required | Nullable       | Defined by                                                                                                                                                                              |
| :-------------------- | --------- | -------- | -------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [name](#name)         | `string`  | Required | cannot be null | [AppConfig](schema-definitions-dependencyendpoint-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/name")         |
| [hostname](#hostname) | `string`  | Required | cannot be null | [AppConfig](schema-definitions-dependencyendpoint-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/hostname") |
| [port](#port)         | `integer` | Required | cannot be null | [AppConfig](schema-definitions-dependencyendpoint-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/port")         |
| [app](#app)           | `string`  | Required | cannot be null | [AppConfig](schema-definitions-dependencyendpoint-properties-app.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/app")           |
| [tlsPort](#tlsport)   | `integer` | Optional | cannot be null | [AppConfig](schema-definitions-dependencyendpoint-properties-tlsport.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/tlsPort")   |

## name

The PodSpec name of the dependent service inside the ClowdApp.


`name`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-dependencyendpoint-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/name")

### name Type

`string`

## hostname

The hostname of the dependent service.


`hostname`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-dependencyendpoint-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/hostname")

### hostname Type

`string`

## port

The port of the dependent service.


`port`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-dependencyendpoint-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/port")

### port Type

`integer`

## app

The app name of the ClowdApp hosting the service.


`app`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-dependencyendpoint-properties-app.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/app")

### app Type

`string`

## tlsPort

The TLS port of the dependent service.


`tlsPort`

-   is optional
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-dependencyendpoint-properties-tlsport.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/DependencyEndpoint/properties/tlsPort")

### tlsPort Type

`integer`
