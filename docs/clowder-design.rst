Clowder Design
==============

Problem Statement
-----------------

Engineers have a steep learning curve when they start writing their first
application for cloud.redhat.com, particularly if they're unfamiliar working in
a cloud-native environment before.

Additionally, enforcing standards and consistency across cloud.redhat.com
applications is difficult because of the number of disparate teams across the
enterprise that contribute to the ecosystem.

Similar to application consistency, it is difficult for engineers to be able to
handle the inconsistencies between environments, e.g local development versus
production.  While these differences can never be fully eliminated, ideally
developers would not have to devise their own solutions to handle these
differences.

Mission
-------

Clowder aims to abstract the underlying Kubernetes environment and configuration
to simplify the creation and maintenance of applications on cloud.redhat.com.

Goals
-----

- Abstract Kubernetes environment and configuration details from applications
- Enable engineers to deploy their application in every Kubernetes environment
  (i.e. dev, stage, prod) with minimal changes to their custom resources.
- Increase operational consistency between cloud.redhat.com applications,
  including rollout parameters and pod affinity.
- Handle metrics configuration and standard SLI/SLO metrics and alerts
- Some form of progressive deployment

Non-goals
---------

- Support applications outside cloud.redhat.com

Proposal
--------

Build a single operator that handles the common use cases that cloud.redhat.com
applications have.  These use cases will be encoded into the API of the
operator, which is of course CRDs.  There will be two CRDs:

1. ``ClowdEnvironment``

   This CR represents an instance of the entire cloud.redhat.com environment,
   e.g. stage or prod.  It contains configuration for various aspects of the
   environment, implemented by *providers*.

2. ``ClowdApp``

   This CR represents a all the configuration an app needs to be deployed into
   the cloud.redhat.com environment, including:

   - One or more deployment specs
   - Kafka topics
   - Databases
   - Object store buckets (e.g. S3)
   - In-memory DBs (e.g. Redis)
   - Public API endpoint name(s)
   - SLO/SLI thresholds


How these CRs will be translated into lower level resource types:

.. image:: images/clowder-flow.svg

Apps will consume their environmental configuration from a JSON document mounted
in their app container.  This JSON document contains the various configuration
that could be considered common across the platform or common kinds of resources
that would be requested by an app on the platform, including:

- Kafka topics
- Connection information for Kafka, object storage, databases, and in-memory DBs
- Port numbers for metrics and webservices
- Public API endpoint prefix

Common configuration interface:

.. image:: images/clowder-new.svg

Alternatives
------------

1. One operator per app, perhaps with a shared library between them

   While this proposal falls neatly in the "app teams are responsible for the
   operation of their app" mantra, the overhead of having each team build and
   maintain their own operator is considered too high.  The design proposed in
   this document helps draw the boundary between the responsibilities of the
   Cloud Dot platform team and the teams that build apps on that platform.

2. One operator with a CRD for each app

   While this would provide significantly more flexibility for teams to
   configure their apps, many app teams would not add any custom attributes,
   thus essentially making their custom CRDs boilerplate code.  In addition,
   teams that did in fact add custom fields would then have to maintain the
   code required to reconcile them.

   The design proposed in this document is intentionally opinionated and makes
   choices on behalf of apps because more often than not dev teams do not have
   strong preferences on many operational aspects of their application.

.. vim: tw=80
