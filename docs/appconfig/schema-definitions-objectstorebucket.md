# Untitled object in AppConfig Schema

```txt
https://cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreConfig/properties/buckets/items
```

Object Storage Bucket


| Abstract            | Extensible | Status         | Identifiable | Custom Properties | Additional Properties | Access Restrictions | Defined In                                                    |
| :------------------ | ---------- | -------------- | ------------ | :---------------- | --------------------- | ------------------- | ------------------------------------------------------------- |
| Can be instantiated | No         | Unknown status | No           | Forbidden         | Allowed               | none                | [schema.json\*](../../out/schema.json "open original schema") |

## items Type

`object` ([Details](schema-definitions-objectstorebucket.md))

# undefined Properties

| Property                        | Type      | Required | Nullable       | Defined by                                                                                                                                                                                      |
| :------------------------------ | --------- | -------- | -------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [accessKey](#accesskey)         | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-objectstorebucket-properties-accesskey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/accessKey")         |
| [secretKey](#secretkey)         | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-objectstorebucket-properties-secretkey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/secretKey")         |
| [region](#region)               | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-objectstorebucket-properties-region.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/region")               |
| [requestedName](#requestedname) | `string`  | Required | cannot be null | [AppConfig](schema-definitions-objectstorebucket-properties-requestedname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/requestedName") |
| [name](#name)                   | `string`  | Required | cannot be null | [AppConfig](schema-definitions-objectstorebucket-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/name")                   |
| [tls](#tls)                     | `boolean` | Optional | cannot be null | [AppConfig](schema-definitions-objectstorebucket-properties-tls.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/tls")                     |
| [endpoint](#endpoint)           | `string`  | Optional | cannot be null | [AppConfig](schema-definitions-objectstorebucket-properties-endpoint.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/endpoint")           |

## accessKey

Defines the access key for specificed bucket.


`accessKey`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstorebucket-properties-accesskey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/accessKey")

### accessKey Type

`string`

## secretKey

Defines the secret key for the specified bucket.


`secretKey`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstorebucket-properties-secretkey.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/secretKey")

### secretKey Type

`string`

## region

Defines the region for the specified bucket.


`region`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstorebucket-properties-region.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/region")

### region Type

`string`

## requestedName

The name that was requested for the bucket in the ClowdApp.


`requestedName`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstorebucket-properties-requestedname.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/requestedName")

### requestedName Type

`string`

## name

The actual name of the bucket being accessed.


`name`

-   is required
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstorebucket-properties-name.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/name")

### name Type

`string`

## tls

Details if the Object Server uses TLS.


`tls`

-   is optional
-   Type: `boolean`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstorebucket-properties-tls.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/tls")

### tls Type

`boolean`

## endpoint

Defines the endpoint for the Object Storage server configuration.


`endpoint`

-   is optional
-   Type: `string`
-   cannot be null
-   defined in: [AppConfig](schema-definitions-objectstorebucket-properties-endpoint.md "https&#x3A;//cloud.redhat.com/schemas/clowder-appconfig#/definitions/ObjectStoreBucket/properties/endpoint")

### endpoint Type

`string`
