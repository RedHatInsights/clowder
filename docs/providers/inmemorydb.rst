..  _inmemorydbprovider:

In-Memory DB Provider
=====================

The **In-Memory DB Provider** is responsible for providing access to an in-memory 
DB instance.

ClowdApp Configuration
----------------------

To request an in-memory db instance, a ``ClowdApp`` would use the `inMemoryDb` stanza, a
partial example of which is shown below.

.. code-block:: yaml

    apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdApp
    metadata:
      name: myapp
    spec:
      # Other App Config
      inMemoryDb: true

ClowdEnv Configuration
----------------------

Modes
*****

The **In-Memory DB Provider** will run in one of the following modes. These are set up by
the ClowdEnvironment. Depending on the environment you are running you may or
may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

redis
^^^^^

In redis mode, the **In-Memory DB Provider** will provision a single node redis instance
in the same namespace as the ``ClowdApp`` that requested it.

ClowdEnv Config options available:

- ``pvc``

elasticache
^^^^^^^^^^^

In elasticache mode, the **In-Memory DB Provider** will search for a secret named
``in-memory-db`` inside the same namespace as the ``ClowdApp`` that requested it.
The hostname and port will then be passed to the ``cdappconfig.json`` for use
by the app.

Generated App Configuration
---------------------------

The In-Memory DB configuration appears in the cdappconfig.json with the following
structure.

JSON structure
**************

.. code-block:: json

    "inMemoryDb": {
      "hostname": "hostname",
      "port": 27015,
      "username": "username",
      "password": "password"
    }

Client access
*************

For supported languages, the kafka configuration is access via the following
attribute names.

+-----------+-----------------------------+
| Language  | Attribute Name              |
+===========+=============================+
| Python    | ``LoadedConfig.inMemoryDb`` |
+-----------+-----------------------------+
| Go        | ``LoadedConfig.InMemoryDb`` |
+-----------+-----------------------------+
| Javscript | ``LoadedConfig.inMemoryDb`` |
+-----------+-----------------------------+
| Ruby      | ``LoadedConfig.inMemoryDb`` |
+-----------+-----------------------------+

ClowdEnv Configuration
**********************

Configuring the **In-Memory DB Provider** is done by providing the follow JSON
structure to the ``ClowdEnv`` resource. Further details of the options
available can be found in the API reference. A minimal example is shown below
for the ``operator`` mode. Different modes can use different configuration
options, more information can be found in the API reference.

.. code-block:: yaml

    apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdEnvivonment
    metadata:
      name: myenv
    spec:
      # Other Env Config
      providers:
        inMemoryDb:
          mode: redis
          pvc: false
