# AppConfig

- [1. Property `root > privatePort`](#privatePort)
- [2. Property `root > publicPort`](#publicPort)
- [3. Property `root > h2cPrivatePort`](#h2cPrivatePort)
- [4. Property `root > h2cPublicPort`](#h2cPublicPort)
- [5. Property `root > webPort`](#webPort)
- [6. Property `root > tlsCAPath`](#tlsCAPath)
- [7. Property `root > metricsPort`](#metricsPort)
- [8. Property `root > metricsPath`](#metricsPath)
- [9. Property `root > logging`](#logging)
  - [9.1. Property `root > logging > type`](#logging_type)
  - [9.2. Property `root > logging > cloudwatch`](#logging_cloudwatch)
    - [9.2.1. Property `root > logging > cloudwatch > accessKeyId`](#logging_cloudwatch_accessKeyId)
    - [9.2.2. Property `root > logging > cloudwatch > secretAccessKey`](#logging_cloudwatch_secretAccessKey)
    - [9.2.3. Property `root > logging > cloudwatch > region`](#logging_cloudwatch_region)
    - [9.2.4. Property `root > logging > cloudwatch > logGroup`](#logging_cloudwatch_logGroup)
- [10. Property `root > metadata`](#metadata)
  - [10.1. Property `root > metadata > name`](#metadata_name)
  - [10.2. Property `root > metadata > envName`](#metadata_envName)
  - [10.3. Property `root > metadata > deployments`](#metadata_deployments)
    - [10.3.1. root > metadata > deployments > DeploymentMetadata](#metadata_deployments_items)
      - [10.3.1.1. Property `root > metadata > deployments > DeploymentMetadata > name`](#metadata_deployments_items_name)
      - [10.3.1.2. Property `root > metadata > deployments > DeploymentMetadata > image`](#metadata_deployments_items_image)
- [11. Property `root > kafka`](#kafka)
  - [11.1. Property `root > kafka > brokers`](#kafka_brokers)
    - [11.1.1. root > kafka > brokers > BrokerConfig](#kafka_brokers_items)
      - [11.1.1.1. Property `root > kafka > brokers > brokers items > hostname`](#kafka_brokers_items_hostname)
      - [11.1.1.2. Property `root > kafka > brokers > brokers items > port`](#kafka_brokers_items_port)
      - [11.1.1.3. Property `root > kafka > brokers > brokers items > cacert`](#kafka_brokers_items_cacert)
      - [11.1.1.4. Property `root > kafka > brokers > brokers items > authtype`](#kafka_brokers_items_authtype)
      - [11.1.1.5. Property `root > kafka > brokers > brokers items > sasl`](#kafka_brokers_items_sasl)
        - [11.1.1.5.1. Property `root > kafka > brokers > brokers items > sasl > username`](#kafka_brokers_items_sasl_username)
        - [11.1.1.5.2. Property `root > kafka > brokers > brokers items > sasl > password`](#kafka_brokers_items_sasl_password)
        - [11.1.1.5.3. Property `root > kafka > brokers > brokers items > sasl > securityProtocol`](#kafka_brokers_items_sasl_securityProtocol)
        - [11.1.1.5.4. Property `root > kafka > brokers > brokers items > sasl > saslMechanism`](#kafka_brokers_items_sasl_saslMechanism)
      - [11.1.1.6. Property `root > kafka > brokers > brokers items > securityProtocol`](#kafka_brokers_items_securityProtocol)
  - [11.2. Property `root > kafka > topics`](#kafka_topics)
    - [11.2.1. root > kafka > topics > TopicConfig](#kafka_topics_items)
      - [11.2.1.1. Property `root > kafka > topics > topics items > requestedName`](#kafka_topics_items_requestedName)
      - [11.2.1.2. Property `root > kafka > topics > topics items > name`](#kafka_topics_items_name)
- [12. Property `root > database`](#database)
  - [12.1. Property `root > database > name`](#database_name)
  - [12.2. Property `root > database > username`](#database_username)
  - [12.3. Property `root > database > password`](#database_password)
  - [12.4. Property `root > database > hostname`](#database_hostname)
  - [12.5. Property `root > database > port`](#database_port)
  - [12.6. Property `root > database > adminUsername`](#database_adminUsername)
  - [12.7. Property `root > database > adminPassword`](#database_adminPassword)
  - [12.8. Property `root > database > rdsCa`](#database_rdsCa)
  - [12.9. Property `root > database > sslMode`](#database_sslMode)
- [13. Property `root > objectStore`](#objectStore)
  - [13.1. Property `root > objectStore > buckets`](#objectStore_buckets)
    - [13.1.1. root > objectStore > buckets > ObjectStoreBucket](#objectStore_buckets_items)
      - [13.1.1.1. Property `root > objectStore > buckets > buckets items > accessKey`](#objectStore_buckets_items_accessKey)
      - [13.1.1.2. Property `root > objectStore > buckets > buckets items > secretKey`](#objectStore_buckets_items_secretKey)
      - [13.1.1.3. Property `root > objectStore > buckets > buckets items > region`](#objectStore_buckets_items_region)
      - [13.1.1.4. Property `root > objectStore > buckets > buckets items > requestedName`](#objectStore_buckets_items_requestedName)
      - [13.1.1.5. Property `root > objectStore > buckets > buckets items > name`](#objectStore_buckets_items_name)
      - [13.1.1.6. Property `root > objectStore > buckets > buckets items > tls`](#objectStore_buckets_items_tls)
      - [13.1.1.7. Property `root > objectStore > buckets > buckets items > endpoint`](#objectStore_buckets_items_endpoint)
  - [13.2. Property `root > objectStore > accessKey`](#objectStore_accessKey)
  - [13.3. Property `root > objectStore > secretKey`](#objectStore_secretKey)
  - [13.4. Property `root > objectStore > hostname`](#objectStore_hostname)
  - [13.5. Property `root > objectStore > port`](#objectStore_port)
  - [13.6. Property `root > objectStore > tls`](#objectStore_tls)
- [14. Property `root > inMemoryDb`](#inMemoryDb)
  - [14.1. Property `root > inMemoryDb > hostname`](#inMemoryDb_hostname)
  - [14.2. Property `root > inMemoryDb > port`](#inMemoryDb_port)
  - [14.3. Property `root > inMemoryDb > username`](#inMemoryDb_username)
  - [14.4. Property `root > inMemoryDb > password`](#inMemoryDb_password)
  - [14.5. Property `root > inMemoryDb > sslMode`](#inMemoryDb_sslMode)
- [15. Property `root > featureFlags`](#featureFlags)
  - [15.1. Property `root > featureFlags > hostname`](#featureFlags_hostname)
  - [15.2. Property `root > featureFlags > port`](#featureFlags_port)
  - [15.3. Property `root > featureFlags > clientAccessToken`](#featureFlags_clientAccessToken)
  - [15.4. Property `root > featureFlags > scheme`](#featureFlags_scheme)
- [16. Property `root > endpoints`](#endpoints)
  - [16.1. root > endpoints > DependencyEndpoint](#endpoints_items)
    - [16.1.1. Property `root > endpoints > endpoints items > name`](#endpoints_items_name)
    - [16.1.2. Property `root > endpoints > endpoints items > hostname`](#endpoints_items_hostname)
    - [16.1.3. Property `root > endpoints > endpoints items > port`](#endpoints_items_port)
    - [16.1.4. Property `root > endpoints > endpoints items > app`](#endpoints_items_app)
    - [16.1.5. Property `root > endpoints > endpoints items > tlsPort`](#endpoints_items_tlsPort)
    - [16.1.6. Property `root > endpoints > endpoints items > h2cPort`](#endpoints_items_h2cPort)
    - [16.1.7. Property `root > endpoints > endpoints items > h2cTLSPort`](#endpoints_items_h2cTLSPort)
    - [16.1.8. Property `root > endpoints > endpoints items > apiPath`](#endpoints_items_apiPath)
    - [16.1.9. Property `root > endpoints > endpoints items > apiPaths`](#endpoints_items_apiPaths)
      - [16.1.9.1. root > endpoints > endpoints items > apiPaths > apiPaths items](#endpoints_items_apiPaths_items)
- [17. Property `root > privateEndpoints`](#privateEndpoints)
  - [17.1. root > privateEndpoints > PrivateDependencyEndpoint](#privateEndpoints_items)
    - [17.1.1. Property `root > privateEndpoints > privateEndpoints items > name`](#privateEndpoints_items_name)
    - [17.1.2. Property `root > privateEndpoints > privateEndpoints items > hostname`](#privateEndpoints_items_hostname)
    - [17.1.3. Property `root > privateEndpoints > privateEndpoints items > port`](#privateEndpoints_items_port)
    - [17.1.4. Property `root > privateEndpoints > privateEndpoints items > app`](#privateEndpoints_items_app)
    - [17.1.5. Property `root > privateEndpoints > privateEndpoints items > tlsPort`](#privateEndpoints_items_tlsPort)
    - [17.1.6. Property `root > privateEndpoints > privateEndpoints items > h2cPort`](#privateEndpoints_items_h2cPort)
    - [17.1.7. Property `root > privateEndpoints > privateEndpoints items > h2cTLSPort`](#privateEndpoints_items_h2cTLSPort)
- [18. Property `root > BOPURL`](#BOPURL)
- [19. Property `root > hashCache`](#hashCache)
- [20. Property `root > hostname`](#hostname)
- [21. Property `root > prometheusGateway`](#prometheusGateway)
  - [21.1. Property `root > prometheusGateway > hostname`](#prometheusGateway_hostname)
  - [21.2. Property `root > prometheusGateway > port`](#prometheusGateway_port)

**Title:** AppConfig

|                           |                         |
| ------------------------- | ----------------------- |
| **Type**                  | `object`                |
| **Required**              | No                      |
| **Additional properties** | Any type allowed        |
| **Defined in**            | #/definitions/AppConfig |

**Description:** ClowdApp deployment configuration for Clowder enabled apps.

| Property                                   | Pattern | Type    | Deprecated | Definition                               | Title/Description                                                                                         |
| ------------------------------------------ | ------- | ------- | ---------- | ---------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| - [privatePort](#privatePort )             | No      | integer | No         | -                                        | Defines the private port that the app should be configured to listen on for API traffic.                  |
| - [publicPort](#publicPort )               | No      | integer | No         | -                                        | Defines the public port that the app should be configured to listen on for API traffic.                   |
| - [h2cPrivatePort](#h2cPrivatePort )       | No      | integer | No         | -                                        | Defines the private H2C port that the app should be configured to listen on for H2C traffic.              |
| - [h2cPublicPort](#h2cPublicPort )         | No      | integer | No         | -                                        | Defines the public H2C port that the app should be configured to listen on for H2C traffic.               |
| - [webPort](#webPort )                     | No      | integer | No         | -                                        | Deprecated: Use 'publicPort' instead.                                                                     |
| - [tlsCAPath](#tlsCAPath )                 | No      | string  | No         | -                                        | Defines the port CA path                                                                                  |
| + [metricsPort](#metricsPort )             | No      | integer | No         | -                                        | Defines the metrics port that the app should be configured to listen on for metric traffic.               |
| + [metricsPath](#metricsPath )             | No      | string  | No         | -                                        | Defines the path to the metrics server that the app should be configured to listen on for metric traffic. |
| + [logging](#logging )                     | No      | object  | No         | In #/definitions/LoggingConfig           | LoggingConfig                                                                                             |
| - [metadata](#metadata )                   | No      | object  | No         | In #/definitions/AppMetadata             | AppMetadata                                                                                               |
| - [kafka](#kafka )                         | No      | object  | No         | In #/definitions/KafkaConfig             | Kafka Configuration                                                                                       |
| - [database](#database )                   | No      | object  | No         | In #/definitions/DatabaseConfig          | DatabaseConfig                                                                                            |
| - [objectStore](#objectStore )             | No      | object  | No         | In #/definitions/ObjectStoreConfig       | Object Storage Configuration                                                                              |
| - [inMemoryDb](#inMemoryDb )               | No      | object  | No         | In #/definitions/InMemoryDBConfig        | In Memory DB Configuration                                                                                |
| - [featureFlags](#featureFlags )           | No      | object  | No         | In #/definitions/FeatureFlagsConfig      | Feature Flags Configuration                                                                               |
| - [endpoints](#endpoints )                 | No      | array   | No         | -                                        | -                                                                                                         |
| - [privateEndpoints](#privateEndpoints )   | No      | array   | No         | -                                        | -                                                                                                         |
| - [BOPURL](#BOPURL )                       | No      | string  | No         | -                                        | Defines the path to the BOPURL.                                                                           |
| - [hashCache](#hashCache )                 | No      | string  | No         | -                                        | A set of configMap/secret hashes                                                                          |
| - [hostname](#hostname )                   | No      | string  | No         | -                                        | The external hostname of the deployment, where applicable                                                 |
| - [prometheusGateway](#prometheusGateway ) | No      | object  | No         | In #/definitions/PrometheusGatewayConfig | Prometheus Gateway Configuration                                                                          |

## <a name="privatePort"></a>1. Property `root > privatePort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** Defines the private port that the app should be configured to listen on for API traffic.

## <a name="publicPort"></a>2. Property `root > publicPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** Defines the public port that the app should be configured to listen on for API traffic.

## <a name="h2cPrivatePort"></a>3. Property `root > h2cPrivatePort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** Defines the private H2C port that the app should be configured to listen on for H2C traffic.

## <a name="h2cPublicPort"></a>4. Property `root > h2cPublicPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** Defines the public H2C port that the app should be configured to listen on for H2C traffic.

## <a name="webPort"></a>5. Property `root > webPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** Deprecated: Use 'publicPort' instead.

## <a name="tlsCAPath"></a>6. Property `root > tlsCAPath`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the port CA path

## <a name="metricsPort"></a>7. Property `root > metricsPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** Defines the metrics port that the app should be configured to listen on for metric traffic.

## <a name="metricsPath"></a>8. Property `root > metricsPath`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the path to the metrics server that the app should be configured to listen on for metric traffic.

## <a name="logging"></a>9. Property `root > logging`

**Title:** LoggingConfig

|                           |                             |
| ------------------------- | --------------------------- |
| **Type**                  | `object`                    |
| **Required**              | Yes                         |
| **Additional properties** | Any type allowed            |
| **Defined in**            | #/definitions/LoggingConfig |

**Description:** Logging Configuration

| Property                             | Pattern | Type   | Deprecated | Definition                        | Title/Description                         |
| ------------------------------------ | ------- | ------ | ---------- | --------------------------------- | ----------------------------------------- |
| + [type](#logging_type )             | No      | string | No         | -                                 | Defines the type of logging configuration |
| - [cloudwatch](#logging_cloudwatch ) | No      | object | No         | In #/definitions/CloudWatchConfig | CloudWatchConfig                          |

### <a name="logging_type"></a>9.1. Property `root > logging > type`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the type of logging configuration

### <a name="logging_cloudwatch"></a>9.2. Property `root > logging > cloudwatch`

**Title:** CloudWatchConfig

|                           |                                |
| ------------------------- | ------------------------------ |
| **Type**                  | `object`                       |
| **Required**              | No                             |
| **Additional properties** | Any type allowed               |
| **Defined in**            | #/definitions/CloudWatchConfig |

**Description:** Cloud Watch configuration

| Property                                                  | Pattern | Type   | Deprecated | Definition | Title/Description                                                          |
| --------------------------------------------------------- | ------- | ------ | ---------- | ---------- | -------------------------------------------------------------------------- |
| + [accessKeyId](#logging_cloudwatch_accessKeyId )         | No      | string | No         | -          | Defines the access key that the app should use for configuring CloudWatch. |
| + [secretAccessKey](#logging_cloudwatch_secretAccessKey ) | No      | string | No         | -          | Defines the secret key that the app should use for configuring CloudWatch. |
| + [region](#logging_cloudwatch_region )                   | No      | string | No         | -          | Defines the region that the app should use for configuring CloudWatch.     |
| + [logGroup](#logging_cloudwatch_logGroup )               | No      | string | No         | -          | Defines the logGroup that the app should use for configuring CloudWatch.   |

#### <a name="logging_cloudwatch_accessKeyId"></a>9.2.1. Property `root > logging > cloudwatch > accessKeyId`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the access key that the app should use for configuring CloudWatch.

#### <a name="logging_cloudwatch_secretAccessKey"></a>9.2.2. Property `root > logging > cloudwatch > secretAccessKey`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the secret key that the app should use for configuring CloudWatch.

#### <a name="logging_cloudwatch_region"></a>9.2.3. Property `root > logging > cloudwatch > region`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the region that the app should use for configuring CloudWatch.

#### <a name="logging_cloudwatch_logGroup"></a>9.2.4. Property `root > logging > cloudwatch > logGroup`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the logGroup that the app should use for configuring CloudWatch.

## <a name="metadata"></a>10. Property `root > metadata`

**Title:** AppMetadata

|                           |                           |
| ------------------------- | ------------------------- |
| **Type**                  | `object`                  |
| **Required**              | No                        |
| **Additional properties** | Any type allowed          |
| **Defined in**            | #/definitions/AppMetadata |

**Description:** Arbitrary metadata pertaining to the application application

| Property                                | Pattern | Type   | Deprecated | Definition | Title/Description                                   |
| --------------------------------------- | ------- | ------ | ---------- | ---------- | --------------------------------------------------- |
| - [name](#metadata_name )               | No      | string | No         | -          | Name of the ClowdApp                                |
| - [envName](#metadata_envName )         | No      | string | No         | -          | Name of the ClowdEnvironment this ClowdApp runs in  |
| - [deployments](#metadata_deployments ) | No      | array  | No         | -          | Metadata pertaining to an application's deployments |

### <a name="metadata_name"></a>10.1. Property `root > metadata > name`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Name of the ClowdApp

### <a name="metadata_envName"></a>10.2. Property `root > metadata > envName`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Name of the ClowdEnvironment this ClowdApp runs in

### <a name="metadata_deployments"></a>10.3. Property `root > metadata > deployments`

|              |         |
| ------------ | ------- |
| **Type**     | `array` |
| **Required** | No      |

**Description:** Metadata pertaining to an application's deployments

|                      | Array restrictions |
| -------------------- | ------------------ |
| **Min items**        | N/A                |
| **Max items**        | N/A                |
| **Items unicity**    | False              |
| **Additional items** | False              |
| **Tuple validation** | See below          |

| Each item of this array must be                   | Description         |
| ------------------------------------------------- | ------------------- |
| [DeploymentMetadata](#metadata_deployments_items) | Deployment Metadata |

#### <a name="metadata_deployments_items"></a>10.3.1. root > metadata > deployments > DeploymentMetadata

**Title:** DeploymentMetadata

|                           |                                  |
| ------------------------- | -------------------------------- |
| **Type**                  | `object`                         |
| **Required**              | No                               |
| **Additional properties** | Any type allowed                 |
| **Defined in**            | #/definitions/DeploymentMetadata |

**Description:** Deployment Metadata

| Property                                      | Pattern | Type   | Deprecated | Definition | Title/Description        |
| --------------------------------------------- | ------- | ------ | ---------- | ---------- | ------------------------ |
| + [name](#metadata_deployments_items_name )   | No      | string | No         | -          | Name of deployment       |
| + [image](#metadata_deployments_items_image ) | No      | string | No         | -          | Image used by deployment |

##### <a name="metadata_deployments_items_name"></a>10.3.1.1. Property `root > metadata > deployments > DeploymentMetadata > name`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Name of deployment

##### <a name="metadata_deployments_items_image"></a>10.3.1.2. Property `root > metadata > deployments > DeploymentMetadata > image`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Image used by deployment

## <a name="kafka"></a>11. Property `root > kafka`

|                           |                           |
| ------------------------- | ------------------------- |
| **Type**                  | `object`                  |
| **Required**              | No                        |
| **Additional properties** | Any type allowed          |
| **Defined in**            | #/definitions/KafkaConfig |

**Description:** Kafka Configuration

| Property                     | Pattern | Type  | Deprecated | Definition | Title/Description                                                        |
| ---------------------------- | ------- | ----- | ---------- | ---------- | ------------------------------------------------------------------------ |
| + [brokers](#kafka_brokers ) | No      | array | No         | -          | Defines the brokers the app should connect to for Kafka services.        |
| + [topics](#kafka_topics )   | No      | array | No         | -          | Defines a list of the topic configurations available to the application. |

### <a name="kafka_brokers"></a>11.1. Property `root > kafka > brokers`

|              |         |
| ------------ | ------- |
| **Type**     | `array` |
| **Required** | Yes     |

**Description:** Defines the brokers the app should connect to for Kafka services.

|                      | Array restrictions |
| -------------------- | ------------------ |
| **Min items**        | N/A                |
| **Max items**        | N/A                |
| **Items unicity**    | False              |
| **Additional items** | False              |
| **Tuple validation** | See below          |

| Each item of this array must be      | Description          |
| ------------------------------------ | -------------------- |
| [BrokerConfig](#kafka_brokers_items) | Broker Configuration |

#### <a name="kafka_brokers_items"></a>11.1.1. root > kafka > brokers > BrokerConfig

|                           |                            |
| ------------------------- | -------------------------- |
| **Type**                  | `object`                   |
| **Required**              | No                         |
| **Additional properties** | Any type allowed           |
| **Defined in**            | #/definitions/BrokerConfig |

**Description:** Broker Configuration

| Property                                                     | Pattern | Type             | Deprecated | Definition                       | Title/Description                                                                                      |
| ------------------------------------------------------------ | ------- | ---------------- | ---------- | -------------------------------- | ------------------------------------------------------------------------------------------------------ |
| + [hostname](#kafka_brokers_items_hostname )                 | No      | string           | No         | -                                | Hostname of kafka broker                                                                               |
| - [port](#kafka_brokers_items_port )                         | No      | integer          | No         | -                                | Port of kafka broker                                                                                   |
| - [cacert](#kafka_brokers_items_cacert )                     | No      | string           | No         | -                                | CA certificate trust list for broker in PEM format. If absent, client should use OS default trust list |
| - [authtype](#kafka_brokers_items_authtype )                 | No      | enum (of string) | No         | -                                | -                                                                                                      |
| - [sasl](#kafka_brokers_items_sasl )                         | No      | object           | No         | In #/definitions/KafkaSASLConfig | SASL Configuration for Kafka                                                                           |
| - [securityProtocol](#kafka_brokers_items_securityProtocol ) | No      | string           | No         | -                                | Broker security procotol, expect one of either: SASL_SSL, SSL                                          |

##### <a name="kafka_brokers_items_hostname"></a>11.1.1.1. Property `root > kafka > brokers > brokers items > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Hostname of kafka broker

##### <a name="kafka_brokers_items_port"></a>11.1.1.2. Property `root > kafka > brokers > brokers items > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** Port of kafka broker

##### <a name="kafka_brokers_items_cacert"></a>11.1.1.3. Property `root > kafka > brokers > brokers items > cacert`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** CA certificate trust list for broker in PEM format. If absent, client should use OS default trust list

##### <a name="kafka_brokers_items_authtype"></a>11.1.1.4. Property `root > kafka > brokers > brokers items > authtype`

|              |                    |
| ------------ | ------------------ |
| **Type**     | `enum (of string)` |
| **Required** | No                 |

Must be one of:
* "sasl"

##### <a name="kafka_brokers_items_sasl"></a>11.1.1.5. Property `root > kafka > brokers > brokers items > sasl`

|                           |                               |
| ------------------------- | ----------------------------- |
| **Type**                  | `object`                      |
| **Required**              | No                            |
| **Additional properties** | Any type allowed              |
| **Defined in**            | #/definitions/KafkaSASLConfig |

**Description:** SASL Configuration for Kafka

| Property                                                          | Pattern | Type   | Deprecated | Definition | Title/Description                                                                                                           |
| ----------------------------------------------------------------- | ------- | ------ | ---------- | ---------- | --------------------------------------------------------------------------------------------------------------------------- |
| - [username](#kafka_brokers_items_sasl_username )                 | No      | string | No         | -          | Broker SASL username                                                                                                        |
| - [password](#kafka_brokers_items_sasl_password )                 | No      | string | No         | -          | Broker SASL password                                                                                                        |
| - [securityProtocol](#kafka_brokers_items_sasl_securityProtocol ) | No      | string | No         | -          | Broker security protocol, expect one of either: SASL_SSL, SSL. DEPRECATED, use the top level securityProtocol field instead |
| - [saslMechanism](#kafka_brokers_items_sasl_saslMechanism )       | No      | string | No         | -          | Broker SASL mechanism, expect: SCRAM-SHA-512                                                                                |

###### <a name="kafka_brokers_items_sasl_username"></a>11.1.1.5.1. Property `root > kafka > brokers > brokers items > sasl > username`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Broker SASL username

###### <a name="kafka_brokers_items_sasl_password"></a>11.1.1.5.2. Property `root > kafka > brokers > brokers items > sasl > password`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Broker SASL password

###### <a name="kafka_brokers_items_sasl_securityProtocol"></a>11.1.1.5.3. Property `root > kafka > brokers > brokers items > sasl > securityProtocol`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Broker security protocol, expect one of either: SASL_SSL, SSL. DEPRECATED, use the top level securityProtocol field instead

###### <a name="kafka_brokers_items_sasl_saslMechanism"></a>11.1.1.5.4. Property `root > kafka > brokers > brokers items > sasl > saslMechanism`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Broker SASL mechanism, expect: SCRAM-SHA-512

##### <a name="kafka_brokers_items_securityProtocol"></a>11.1.1.6. Property `root > kafka > brokers > brokers items > securityProtocol`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Broker security procotol, expect one of either: SASL_SSL, SSL

### <a name="kafka_topics"></a>11.2. Property `root > kafka > topics`

|              |         |
| ------------ | ------- |
| **Type**     | `array` |
| **Required** | Yes     |

**Description:** Defines a list of the topic configurations available to the application.

|                      | Array restrictions |
| -------------------- | ------------------ |
| **Min items**        | N/A                |
| **Max items**        | N/A                |
| **Items unicity**    | False              |
| **Additional items** | False              |
| **Tuple validation** | See below          |

| Each item of this array must be    | Description         |
| ---------------------------------- | ------------------- |
| [TopicConfig](#kafka_topics_items) | Topic Configuration |

#### <a name="kafka_topics_items"></a>11.2.1. root > kafka > topics > TopicConfig

|                           |                           |
| ------------------------- | ------------------------- |
| **Type**                  | `object`                  |
| **Required**              | No                        |
| **Additional properties** | Any type allowed          |
| **Defined in**            | #/definitions/TopicConfig |

**Description:** Topic Configuration

| Property                                              | Pattern | Type   | Deprecated | Definition | Title/Description                                           |
| ----------------------------------------------------- | ------- | ------ | ---------- | ---------- | ----------------------------------------------------------- |
| + [requestedName](#kafka_topics_items_requestedName ) | No      | string | No         | -          | The name that the app requested in the ClowdApp definition. |
| + [name](#kafka_topics_items_name )                   | No      | string | No         | -          | The name of the actual topic on the Kafka server.           |

##### <a name="kafka_topics_items_requestedName"></a>11.2.1.1. Property `root > kafka > topics > topics items > requestedName`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The name that the app requested in the ClowdApp definition.

##### <a name="kafka_topics_items_name"></a>11.2.1.2. Property `root > kafka > topics > topics items > name`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The name of the actual topic on the Kafka server.

## <a name="database"></a>12. Property `root > database`

**Title:** DatabaseConfig

|                           |                              |
| ------------------------- | ---------------------------- |
| **Type**                  | `object`                     |
| **Required**              | No                           |
| **Additional properties** | Any type allowed             |
| **Defined in**            | #/definitions/DatabaseConfig |

**Description:** Database Configuration

| Property                                    | Pattern | Type    | Deprecated | Definition | Title/Description                                                 |
| ------------------------------------------- | ------- | ------- | ---------- | ---------- | ----------------------------------------------------------------- |
| + [name](#database_name )                   | No      | string  | No         | -          | Defines the database name.                                        |
| + [username](#database_username )           | No      | string  | No         | -          | Defines a username with standard access to the database.          |
| + [password](#database_password )           | No      | string  | No         | -          | Defines the password for the standard user.                       |
| + [hostname](#database_hostname )           | No      | string  | No         | -          | Defines the hostname of the database configured for the ClowdApp. |
| + [port](#database_port )                   | No      | integer | No         | -          | Defines the port of the database configured for the ClowdApp.     |
| + [adminUsername](#database_adminUsername ) | No      | string  | No         | -          | Defines the pgAdmin username.                                     |
| + [adminPassword](#database_adminPassword ) | No      | string  | No         | -          | Defines the pgAdmin password.                                     |
| - [rdsCa](#database_rdsCa )                 | No      | string  | No         | -          | Defines the CA used to access the database.                       |
| + [sslMode](#database_sslMode )             | No      | string  | No         | -          | Defines the postgres SSL mode that should be used.                |

### <a name="database_name"></a>12.1. Property `root > database > name`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the database name.

### <a name="database_username"></a>12.2. Property `root > database > username`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines a username with standard access to the database.

### <a name="database_password"></a>12.3. Property `root > database > password`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the password for the standard user.

### <a name="database_hostname"></a>12.4. Property `root > database > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the hostname of the database configured for the ClowdApp.

### <a name="database_port"></a>12.5. Property `root > database > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** Defines the port of the database configured for the ClowdApp.

### <a name="database_adminUsername"></a>12.6. Property `root > database > adminUsername`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the pgAdmin username.

### <a name="database_adminPassword"></a>12.7. Property `root > database > adminPassword`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the pgAdmin password.

### <a name="database_rdsCa"></a>12.8. Property `root > database > rdsCa`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the CA used to access the database.

### <a name="database_sslMode"></a>12.9. Property `root > database > sslMode`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the postgres SSL mode that should be used.

## <a name="objectStore"></a>13. Property `root > objectStore`

|                           |                                 |
| ------------------------- | ------------------------------- |
| **Type**                  | `object`                        |
| **Required**              | No                              |
| **Additional properties** | Any type allowed                |
| **Defined in**            | #/definitions/ObjectStoreConfig |

**Description:** Object Storage Configuration

| Property                               | Pattern | Type    | Deprecated | Definition | Title/Description                                                   |
| -------------------------------------- | ------- | ------- | ---------- | ---------- | ------------------------------------------------------------------- |
| - [buckets](#objectStore_buckets )     | No      | array   | No         | -          | -                                                                   |
| - [accessKey](#objectStore_accessKey ) | No      | string  | No         | -          | Defines the access key for the Object Storage server configuration. |
| - [secretKey](#objectStore_secretKey ) | No      | string  | No         | -          | Defines the secret key for the Object Storage server configuration. |
| + [hostname](#objectStore_hostname )   | No      | string  | No         | -          | Defines the hostname for the Object Storage server configuration.   |
| + [port](#objectStore_port )           | No      | integer | No         | -          | Defines the port for the Object Storage server configuration.       |
| + [tls](#objectStore_tls )             | No      | boolean | No         | -          | Details if the Object Server uses TLS.                              |

### <a name="objectStore_buckets"></a>13.1. Property `root > objectStore > buckets`

|              |         |
| ------------ | ------- |
| **Type**     | `array` |
| **Required** | No      |

|                      | Array restrictions |
| -------------------- | ------------------ |
| **Min items**        | N/A                |
| **Max items**        | N/A                |
| **Items unicity**    | False              |
| **Additional items** | False              |
| **Tuple validation** | See below          |

| Each item of this array must be                 | Description           |
| ----------------------------------------------- | --------------------- |
| [ObjectStoreBucket](#objectStore_buckets_items) | Object Storage Bucket |

#### <a name="objectStore_buckets_items"></a>13.1.1. root > objectStore > buckets > ObjectStoreBucket

|                           |                                 |
| ------------------------- | ------------------------------- |
| **Type**                  | `object`                        |
| **Required**              | No                              |
| **Additional properties** | Any type allowed                |
| **Defined in**            | #/definitions/ObjectStoreBucket |

**Description:** Object Storage Bucket

| Property                                                     | Pattern | Type    | Deprecated | Definition | Title/Description                                                 |
| ------------------------------------------------------------ | ------- | ------- | ---------- | ---------- | ----------------------------------------------------------------- |
| - [accessKey](#objectStore_buckets_items_accessKey )         | No      | string  | No         | -          | Defines the access key for specificed bucket.                     |
| - [secretKey](#objectStore_buckets_items_secretKey )         | No      | string  | No         | -          | Defines the secret key for the specified bucket.                  |
| - [region](#objectStore_buckets_items_region )               | No      | string  | No         | -          | Defines the region for the specified bucket.                      |
| + [requestedName](#objectStore_buckets_items_requestedName ) | No      | string  | No         | -          | The name that was requested for the bucket in the ClowdApp.       |
| + [name](#objectStore_buckets_items_name )                   | No      | string  | No         | -          | The actual name of the bucket being accessed.                     |
| - [tls](#objectStore_buckets_items_tls )                     | No      | boolean | No         | -          | Details if the Object Server uses TLS.                            |
| - [endpoint](#objectStore_buckets_items_endpoint )           | No      | string  | No         | -          | Defines the endpoint for the Object Storage server configuration. |

##### <a name="objectStore_buckets_items_accessKey"></a>13.1.1.1. Property `root > objectStore > buckets > buckets items > accessKey`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the access key for specificed bucket.

##### <a name="objectStore_buckets_items_secretKey"></a>13.1.1.2. Property `root > objectStore > buckets > buckets items > secretKey`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the secret key for the specified bucket.

##### <a name="objectStore_buckets_items_region"></a>13.1.1.3. Property `root > objectStore > buckets > buckets items > region`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the region for the specified bucket.

##### <a name="objectStore_buckets_items_requestedName"></a>13.1.1.4. Property `root > objectStore > buckets > buckets items > requestedName`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The name that was requested for the bucket in the ClowdApp.

##### <a name="objectStore_buckets_items_name"></a>13.1.1.5. Property `root > objectStore > buckets > buckets items > name`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The actual name of the bucket being accessed.

##### <a name="objectStore_buckets_items_tls"></a>13.1.1.6. Property `root > objectStore > buckets > buckets items > tls`

|              |           |
| ------------ | --------- |
| **Type**     | `boolean` |
| **Required** | No        |

**Description:** Details if the Object Server uses TLS.

##### <a name="objectStore_buckets_items_endpoint"></a>13.1.1.7. Property `root > objectStore > buckets > buckets items > endpoint`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the endpoint for the Object Storage server configuration.

### <a name="objectStore_accessKey"></a>13.2. Property `root > objectStore > accessKey`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the access key for the Object Storage server configuration.

### <a name="objectStore_secretKey"></a>13.3. Property `root > objectStore > secretKey`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the secret key for the Object Storage server configuration.

### <a name="objectStore_hostname"></a>13.4. Property `root > objectStore > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the hostname for the Object Storage server configuration.

### <a name="objectStore_port"></a>13.5. Property `root > objectStore > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** Defines the port for the Object Storage server configuration.

### <a name="objectStore_tls"></a>13.6. Property `root > objectStore > tls`

|              |           |
| ------------ | --------- |
| **Type**     | `boolean` |
| **Required** | Yes       |

**Description:** Details if the Object Server uses TLS.

## <a name="inMemoryDb"></a>14. Property `root > inMemoryDb`

|                           |                                |
| ------------------------- | ------------------------------ |
| **Type**                  | `object`                       |
| **Required**              | No                             |
| **Additional properties** | Any type allowed               |
| **Defined in**            | #/definitions/InMemoryDBConfig |

**Description:** In Memory DB Configuration

| Property                            | Pattern | Type    | Deprecated | Definition | Title/Description                                                |
| ----------------------------------- | ------- | ------- | ---------- | ---------- | ---------------------------------------------------------------- |
| + [hostname](#inMemoryDb_hostname ) | No      | string  | No         | -          | Defines the hostname for the In Memory DB server configuration.  |
| + [port](#inMemoryDb_port )         | No      | integer | No         | -          | Defines the port for the In Memory DB server configuration.      |
| - [username](#inMemoryDb_username ) | No      | string  | No         | -          | Defines the username for the In Memory DB server configuration.  |
| - [password](#inMemoryDb_password ) | No      | string  | No         | -          | Defines the password for the In Memory DB server configuration.  |
| - [sslMode](#inMemoryDb_sslMode )   | No      | boolean | No         | -          | Defines the sslMode used by the In Memory DB server coniguration |

### <a name="inMemoryDb_hostname"></a>14.1. Property `root > inMemoryDb > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the hostname for the In Memory DB server configuration.

### <a name="inMemoryDb_port"></a>14.2. Property `root > inMemoryDb > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** Defines the port for the In Memory DB server configuration.

### <a name="inMemoryDb_username"></a>14.3. Property `root > inMemoryDb > username`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the username for the In Memory DB server configuration.

### <a name="inMemoryDb_password"></a>14.4. Property `root > inMemoryDb > password`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the password for the In Memory DB server configuration.

### <a name="inMemoryDb_sslMode"></a>14.5. Property `root > inMemoryDb > sslMode`

|              |           |
| ------------ | --------- |
| **Type**     | `boolean` |
| **Required** | No        |

**Description:** Defines the sslMode used by the In Memory DB server coniguration

## <a name="featureFlags"></a>15. Property `root > featureFlags`

|                           |                                  |
| ------------------------- | -------------------------------- |
| **Type**                  | `object`                         |
| **Required**              | No                               |
| **Additional properties** | Any type allowed                 |
| **Defined in**            | #/definitions/FeatureFlagsConfig |

**Description:** Feature Flags Configuration

| Property                                                | Pattern | Type             | Deprecated | Definition | Title/Description                                                              |
| ------------------------------------------------------- | ------- | ---------------- | ---------- | ---------- | ------------------------------------------------------------------------------ |
| + [hostname](#featureFlags_hostname )                   | No      | string           | No         | -          | Defines the hostname for the FeatureFlags server                               |
| + [port](#featureFlags_port )                           | No      | integer          | No         | -          | Defines the port for the FeatureFlags server                                   |
| - [clientAccessToken](#featureFlags_clientAccessToken ) | No      | string           | No         | -          | Defines the client access token to use when connect to the FeatureFlags server |
| + [scheme](#featureFlags_scheme )                       | No      | enum (of string) | No         | -          | Details the scheme to use for FeatureFlags http/https                          |

### <a name="featureFlags_hostname"></a>15.1. Property `root > featureFlags > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the hostname for the FeatureFlags server

### <a name="featureFlags_port"></a>15.2. Property `root > featureFlags > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** Defines the port for the FeatureFlags server

### <a name="featureFlags_clientAccessToken"></a>15.3. Property `root > featureFlags > clientAccessToken`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the client access token to use when connect to the FeatureFlags server

### <a name="featureFlags_scheme"></a>15.4. Property `root > featureFlags > scheme`

|              |                    |
| ------------ | ------------------ |
| **Type**     | `enum (of string)` |
| **Required** | Yes                |

**Description:** Details the scheme to use for FeatureFlags http/https

Must be one of:
* "http"
* "https"

## <a name="endpoints"></a>16. Property `root > endpoints`

|              |         |
| ------------ | ------- |
| **Type**     | `array` |
| **Required** | No      |

|                      | Array restrictions |
| -------------------- | ------------------ |
| **Min items**        | N/A                |
| **Max items**        | N/A                |
| **Items unicity**    | False              |
| **Additional items** | False              |
| **Tuple validation** | See below          |

| Each item of this array must be        | Description                       |
| -------------------------------------- | --------------------------------- |
| [DependencyEndpoint](#endpoints_items) | Dependent service connection info |

### <a name="endpoints_items"></a>16.1. root > endpoints > DependencyEndpoint

|                           |                                  |
| ------------------------- | -------------------------------- |
| **Type**                  | `object`                         |
| **Required**              | No                               |
| **Additional properties** | Any type allowed                 |
| **Defined in**            | #/definitions/DependencyEndpoint |

**Description:** Dependent service connection info

| Property                                     | Pattern | Type            | Deprecated | Definition | Title/Description                                                                                      |
| -------------------------------------------- | ------- | --------------- | ---------- | ---------- | ------------------------------------------------------------------------------------------------------ |
| + [name](#endpoints_items_name )             | No      | string          | No         | -          | The PodSpec name of the dependent service inside the ClowdApp.                                         |
| + [hostname](#endpoints_items_hostname )     | No      | string          | No         | -          | The hostname of the dependent service.                                                                 |
| + [port](#endpoints_items_port )             | No      | integer         | No         | -          | The port of the dependent service.                                                                     |
| + [app](#endpoints_items_app )               | No      | string          | No         | -          | The app name of the ClowdApp hosting the service.                                                      |
| - [tlsPort](#endpoints_items_tlsPort )       | No      | integer         | No         | -          | The TLS port of the dependent service.                                                                 |
| - [h2cPort](#endpoints_items_h2cPort )       | No      | integer         | No         | -          | The H2C port of the dependent service.                                                                 |
| - [h2cTLSPort](#endpoints_items_h2cTLSPort ) | No      | integer         | No         | -          | The H2C TLS port of the dependent service.                                                             |
| + [apiPath](#endpoints_items_apiPath )       | No      | string          | No         | -          | The top level api path that the app should serve from /api/<apiPath> (deprecated, use apiPaths)        |
| - [apiPaths](#endpoints_items_apiPaths )     | No      | array of string | No         | -          | The list of API paths (each matching format: '/api/some-path/') that this app will serve requests from |

#### <a name="endpoints_items_name"></a>16.1.1. Property `root > endpoints > endpoints items > name`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The PodSpec name of the dependent service inside the ClowdApp.

#### <a name="endpoints_items_hostname"></a>16.1.2. Property `root > endpoints > endpoints items > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The hostname of the dependent service.

#### <a name="endpoints_items_port"></a>16.1.3. Property `root > endpoints > endpoints items > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** The port of the dependent service.

#### <a name="endpoints_items_app"></a>16.1.4. Property `root > endpoints > endpoints items > app`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The app name of the ClowdApp hosting the service.

#### <a name="endpoints_items_tlsPort"></a>16.1.5. Property `root > endpoints > endpoints items > tlsPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** The TLS port of the dependent service.

#### <a name="endpoints_items_h2cPort"></a>16.1.6. Property `root > endpoints > endpoints items > h2cPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** The H2C port of the dependent service.

#### <a name="endpoints_items_h2cTLSPort"></a>16.1.7. Property `root > endpoints > endpoints items > h2cTLSPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** The H2C TLS port of the dependent service.

#### <a name="endpoints_items_apiPath"></a>16.1.8. Property `root > endpoints > endpoints items > apiPath`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The top level api path that the app should serve from /api/<apiPath> (deprecated, use apiPaths)

#### <a name="endpoints_items_apiPaths"></a>16.1.9. Property `root > endpoints > endpoints items > apiPaths`

|              |                   |
| ------------ | ----------------- |
| **Type**     | `array of string` |
| **Required** | No                |

**Description:** The list of API paths (each matching format: '/api/some-path/') that this app will serve requests from

|                      | Array restrictions |
| -------------------- | ------------------ |
| **Min items**        | N/A                |
| **Max items**        | N/A                |
| **Items unicity**    | False              |
| **Additional items** | False              |
| **Tuple validation** | See below          |

| Each item of this array must be                   | Description |
| ------------------------------------------------- | ----------- |
| [apiPaths items](#endpoints_items_apiPaths_items) | -           |

##### <a name="endpoints_items_apiPaths_items"></a>16.1.9.1. root > endpoints > endpoints items > apiPaths > apiPaths items

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

## <a name="privateEndpoints"></a>17. Property `root > privateEndpoints`

|              |         |
| ------------ | ------- |
| **Type**     | `array` |
| **Required** | No      |

|                      | Array restrictions |
| -------------------- | ------------------ |
| **Min items**        | N/A                |
| **Max items**        | N/A                |
| **Items unicity**    | False              |
| **Additional items** | False              |
| **Tuple validation** | See below          |

| Each item of this array must be                      | Description                       |
| ---------------------------------------------------- | --------------------------------- |
| [PrivateDependencyEndpoint](#privateEndpoints_items) | Dependent service connection info |

### <a name="privateEndpoints_items"></a>17.1. root > privateEndpoints > PrivateDependencyEndpoint

|                           |                                         |
| ------------------------- | --------------------------------------- |
| **Type**                  | `object`                                |
| **Required**              | No                                      |
| **Additional properties** | Any type allowed                        |
| **Defined in**            | #/definitions/PrivateDependencyEndpoint |

**Description:** Dependent service connection info

| Property                                            | Pattern | Type    | Deprecated | Definition | Title/Description                                              |
| --------------------------------------------------- | ------- | ------- | ---------- | ---------- | -------------------------------------------------------------- |
| + [name](#privateEndpoints_items_name )             | No      | string  | No         | -          | The PodSpec name of the dependent service inside the ClowdApp. |
| + [hostname](#privateEndpoints_items_hostname )     | No      | string  | No         | -          | The hostname of the dependent service.                         |
| + [port](#privateEndpoints_items_port )             | No      | integer | No         | -          | The port of the dependent service.                             |
| + [app](#privateEndpoints_items_app )               | No      | string  | No         | -          | The app name of the ClowdApp hosting the service.              |
| - [tlsPort](#privateEndpoints_items_tlsPort )       | No      | integer | No         | -          | The TLS port of the dependent service.                         |
| - [h2cPort](#privateEndpoints_items_h2cPort )       | No      | integer | No         | -          | The H2C port of the dependent service.                         |
| - [h2cTLSPort](#privateEndpoints_items_h2cTLSPort ) | No      | integer | No         | -          | The H2C TLS port of the dependent service.                     |

#### <a name="privateEndpoints_items_name"></a>17.1.1. Property `root > privateEndpoints > privateEndpoints items > name`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The PodSpec name of the dependent service inside the ClowdApp.

#### <a name="privateEndpoints_items_hostname"></a>17.1.2. Property `root > privateEndpoints > privateEndpoints items > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The hostname of the dependent service.

#### <a name="privateEndpoints_items_port"></a>17.1.3. Property `root > privateEndpoints > privateEndpoints items > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** The port of the dependent service.

#### <a name="privateEndpoints_items_app"></a>17.1.4. Property `root > privateEndpoints > privateEndpoints items > app`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** The app name of the ClowdApp hosting the service.

#### <a name="privateEndpoints_items_tlsPort"></a>17.1.5. Property `root > privateEndpoints > privateEndpoints items > tlsPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** The TLS port of the dependent service.

#### <a name="privateEndpoints_items_h2cPort"></a>17.1.6. Property `root > privateEndpoints > privateEndpoints items > h2cPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** The H2C port of the dependent service.

#### <a name="privateEndpoints_items_h2cTLSPort"></a>17.1.7. Property `root > privateEndpoints > privateEndpoints items > h2cTLSPort`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | No        |

**Description:** The H2C TLS port of the dependent service.

## <a name="BOPURL"></a>18. Property `root > BOPURL`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** Defines the path to the BOPURL.

## <a name="hashCache"></a>19. Property `root > hashCache`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** A set of configMap/secret hashes

## <a name="hostname"></a>20. Property `root > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | No       |

**Description:** The external hostname of the deployment, where applicable

## <a name="prometheusGateway"></a>21. Property `root > prometheusGateway`

|                           |                                       |
| ------------------------- | ------------------------------------- |
| **Type**                  | `object`                              |
| **Required**              | No                                    |
| **Additional properties** | Any type allowed                      |
| **Defined in**            | #/definitions/PrometheusGatewayConfig |

**Description:** Prometheus Gateway Configuration

| Property                                   | Pattern | Type    | Deprecated | Definition | Title/Description                                                     |
| ------------------------------------------ | ------- | ------- | ---------- | ---------- | --------------------------------------------------------------------- |
| + [hostname](#prometheusGateway_hostname ) | No      | string  | No         | -          | Defines the hostname for the Prometheus Gateway server configuration. |
| + [port](#prometheusGateway_port )         | No      | integer | No         | -          | Defines the port for the Prometheus Gateway server configuration.     |

### <a name="prometheusGateway_hostname"></a>21.1. Property `root > prometheusGateway > hostname`

|              |          |
| ------------ | -------- |
| **Type**     | `string` |
| **Required** | Yes      |

**Description:** Defines the hostname for the Prometheus Gateway server configuration.

### <a name="prometheusGateway_port"></a>21.2. Property `root > prometheusGateway > port`

|              |           |
| ------------ | --------- |
| **Type**     | `integer` |
| **Required** | Yes       |

**Description:** Defines the port for the Prometheus Gateway server configuration.

----------------------------------------------------------------------------------------------------------------------------
