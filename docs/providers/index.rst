Providers
=========

Clowder has several different providers that can function in different modes
the following links lead to pages describing these providers and their modes.

Modes are generally configured in a ``ClowdEnvironment`` resource and
``ClowdApp`` resources reference the ``ClowdEnvironment`` they wish to operate
under. For production environments there is usually only one
``ClowdEnvironment`` resource. It is important to understand how these
providers operate in their different modes to know if there are any
prerequisites that need to be created for the ``ClowdApp`` to function
properly.

.. toctree::
    :caption: Reference Guides
    :maxdepth: 1

    confighash
    cronjob
    database
    dependencies
    deployment
    kafka
    featureflags
    inmemorydb
    logging
    metrics
    objectstore
    serviceaccount
    servicemesh
    web
