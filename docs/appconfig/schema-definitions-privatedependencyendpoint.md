# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint
```

Dependent service connection info


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## PrivateDependencyEndpoint Type

`object` ([Details](schema-definitions-privatedependencyendpoint.md))

# undefined Properties

| Property              | Type      | Required | Nullable       | Defined by                                                                                                                                                                                            |
| :-------------------- | --------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [name](#name)         | `string`  | Required | cannot be null | [AppConfig](schema-definitions-privatedependencyendpoint-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/name")         |
| [hostname](#hostname) | `string`  | Required | cannot be null | [AppConfig](schema-definitions-privatedependencyendpoint-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/hostname") |
| [port](#port)         | `integer` | Required | cannot be null | [AppConfig](schema-definitions-privatedependencyendpoint-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/port")         |
| [app](#app)           | `string`  | Required | cannot be null | [AppConfig](schema-definitions-privatedependencyendpoint-properties-app.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/app")           |

## name

The PodSpec name of the dependent service inside the ClowdApp.


`name`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-privatedependencyendpoint-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/name")

### name Type

`string`

## hostname

The hostname of the dependent service.


`hostname`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-privatedependencyendpoint-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/hostname")

### hostname Type

`string`

## port

The port of the dependent service.


`port`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-privatedependencyendpoint-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/port")

### port Type

`integer`

## app

The app name of the ClowdApp hosting the service.


`app`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-privatedependencyendpoint-properties-app.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/PrivateDependencyEndpoint/properties/app")

### app Type

`string`
