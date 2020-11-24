Operating Clowder
=================

Two primary aspects of operating Clowder: Operating the apps managed by Clowder
and operating Clowder itself.

Operating Apps Managed by Clowder
---------------------------------

**ClowdEnvironment**

- Cluster scoped
- Providers

  - Kafka
  - Logging
  - Object storage
  - In-memroy DB
  - Relational DB

- Modes
- Target namespace
- abbreviated env

**ClowdApp**

- Namespace scoped
- Connection to ClowdEnvironment
- Dependencies

  - Services
  - Kafka topics
  - Logging
  - Object storage
  - In-memory DB
  - Relational DB

- Resources created

  - Deployment

    - Anti-affinity
    - Image pull policy
    - Standard ports

  - Service
  - Secret (cdappconfig.json)

- abbreviated app

Operating Clowder Itself
------------------------

- OLM pipeline
- Metrics and alerts
- App-interface modes
