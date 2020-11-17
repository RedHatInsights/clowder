# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig
```

Object Storage Configuration


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                          |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../../../out/schema.json "open original schema") |

## ObjectStoreConfig Type

`object` ([Details](schema-definitions-objectstoreconfig.md))

# undefined Properties

| Property                | Type      | Required | Nullable       | Defined by                                                                                                                                                                              |
| :---------------------- | --------- | -------- | -------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [buckets](#buckets)     | `array`   | Optional | cannot be null | [AppConfig](schema-definitions-objectstoreconfig-properties-buckets.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/buckets")     |
| [accessKey](#accesskey) | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-objectstoreconfig-properties-accesskey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/accessKey") |
| [secretKey](#secretkey) | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-objectstoreconfig-properties-secretkey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/secretKey") |
| [hostname](#hostname)   | `string`  | Required | cannot be null | [AppConfig](schema-definitions-objectstoreconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/hostname")   |
| [port](#port)           | `integer` | Required | cannot be null | [AppConfig](schema-definitions-objectstoreconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/port")           |
| [tls](#tls)             | `boolean` | Required | cannot be null | [AppConfig](schema-definitions-objectstoreconfig-properties-tls.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/tls")             |

## buckets




`buckets`

-   is optional
-   Type: `object[]` ([Details](schema-definitions-objectstorebucket.md))
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstoreconfig-properties-buckets.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/buckets")

### buckets Type

`object[]` ([Details](schema-definitions-objectstorebucket.md))

## accessKey

Defines the access key for the Object Storage server configuration.


`accessKey`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstoreconfig-properties-accesskey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/accessKey")

### accessKey Type

`string`

## secretKey

Defines the secret key for the Object Storage server configuration.


`secretKey`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstoreconfig-properties-secretkey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/secretKey")

### secretKey Type

`string`

## hostname

Defines the hostname for the Object Storage server configuration.


`hostname`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstoreconfig-properties-hostname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/hostname")

### hostname Type

`string`

## port

Defines the port for the Object Storage server configuration.


`port`

-   is required
-   Type: `integer`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstoreconfig-properties-port.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/port")

### port Type

`integer`

## tls

Details if the Object Server uses TLS.


`tls`

-   is required
-   Type: `boolean`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstoreconfig-properties-tls.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/tls")

### tls Type

`boolean`
