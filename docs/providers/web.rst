..  _webprovider:

Web Provider
============

The **Web Provider** is responsible for creating the Service for the public/private port and adding the
port to the container template on the deployment.

ClowdApp Configuration
----------------------

The public and private ports can be enabled by using the ``webServices`` stanza on the ``ClowdApp``
specification.

.. code-block:: yaml

    apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdApp
    metadata:
      name: myapp
    spec:
      # Other App Config
      deployments:
        name: inventory
        podSpec: 
          image: quay.io/psav/clowder-hello
        webServices:
          public:
            enabled: true
          private:
            enabled: true

ClowdEnv Configuration
----------------------

Modes
*****

The **Web Provider** will run in one of the following modes. These are set up
by the ClowdEnvironment. Depending on the environment you are running you may
or may not have access to change this mode. More information on provider
configuration is at the bottom of this page.

operator
^^^^^^^^

In operator mode, the **Web Provider** will set the port and service for a deployment.

ClowdEnv Config options available:

- ``port``
- ``privatePort``
- ``apiPrefix``

Generated App Configuration
---------------------------

The Metrics configuration appears in the cdappconfig.json with the following
structure.

JSON structure
**************

.. code-block:: json

    {
      "publicPort": 8000,
      "privatePort": 10000,
      "apiPrefix": "/api"
    }

Client access
*************

For supported languages, the web configuration is access via the following
attribute names.

+-----------+------------------------------+
| Language  | Attribute Name               |
+===========+==============================+
| Python    | ``LoadedConfig.publicPort``  |
+-----------+------------------------------+
| Go        | ``LoadedConfig.PublicPort``  |
+-----------+------------------------------+
| Javscript | ``LoadedConfig.publicPort``  |
+-----------+------------------------------+
| Ruby      | ``LoadedConfig.publicPort``  |
+-----------+------------------------------+



ClowdEnv Configuration
**********************

The **Web Provider** can be configured to set the public port, private port and path as follows in this 
example.

.. code-block:: yaml

    apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdEnvivonment
    metadata:
      name: myenv
    spec:
      # Other Env Config
      providers:
        web:
          mode: operator
          privatePort: 10000
          port: 8000
