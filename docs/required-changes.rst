Required Changes for Apps
=========================

Deployment and configuration on an app on the Platform becomes much simpler
after migrating to clowder because the operator removes lots of operational
decisions from the app, e.g. logging and kafka topic configuration.  This
simplicity comes at a price, of course:  teams must ensure conformity to the
conventions enforced by the operator before their app can be managed by the
it.

A rough list of changes most app teams will need to make in order to be deployed
by the new operator:

Accept platform configuration from the mounted JSON document
------------------------------------------------------------

* Service hostnames
* Kafka boostrap URL
* Kafka topic names
* Web port number
* Metrics port number and path

Deploy metrics endpoint on isolated port
----------------------------------------

If an app's metrics endpoint is deployed on the same port as a different web
service, it needs to be modified to use a different port.  It is a `best practice`_
to deploy the metrics endpoint on a separate port.

.. _best practice: https://github.com/korfuri/django-prometheus/blob/master/documentation/exports.md#exporting-metrics-in-a-dedicated-thread

Use minio as the only object storage client library
---------------------------------------------------

The operator will use minio to provision buckets in the ephemeral environments,
and the `minio client library`_ supports connecting to Amazon S3. 

.. _minio client library: https://docs.min.io/docs/python-client-api-reference.html

Move deployment template to source code repo
--------------------------------------------

Required by App SRE

Write build_deploy.sh and pr_check.sh
-------------------------------------

Required by App SRE.  ``pr_check.sh`` will be where smoke tests are invoked.

Create build pipeline in app-interface
--------------------------------------

Required by App SRE

Replace Deployment and Service resources with Application
---------------------------------------------------------

This is what actually makes an app managed by the operator.

Apps using OAuth proxy will need to migrate to Turnpike
-------------------------------------------------------

.. vim: tw=80

Standardize on how you configure Redis
--------------------------------------

Create a list of service dependencies
-------------------------------------
