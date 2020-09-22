Insights Operator Design
========================

The goal of this document is to develop a design for an operator that can
deploy and operate one or more instances of the SaaS application known as the
*Insights Platform*.

Why Operators?
==============

An operator is an application that follows various conventions which enable it
to constantly monitor and react to its Openshift environment.

While most operators you may be familiar with are shipped to customers, the
operator we are building will not be shipped to customers; instead it will be
used to operate our SaaS-based application, in tandem with App SRE's
app-interface toolchain.  This allows us greater flexibility in the design and
management of our operator.

There are two high-level aspects of the value provided by an operator for a
SaaS-based service:  deployment configuration and operational resiliency

Deployment Configuration
------------------------

Operators come with their own Custom Resource Definitions (CRDs), which
effectively extend the Kubernetes API.  An example of a resource definition you
are familiar with is the ``DeploymentConfig`` resource type.  This translates
to a simpler set of resources defined for any given app on the Insights
Platform.  Instead of an app having to define a ``Deployment``, ``Service``,
various logging and database secrets, etc., an app just defines a single CRD,
and the operator will take care of creating all of the "low level" resources on
the app's behalf.

Operational Resiliency
----------------------

Operators are able to constantly monitor the state of all the resources it
manages and then respond with the appropriate corrective action when
identifying abnormalities or errors.  Think of it like a set of playbooks or
SOPs codified into the operator's codebase.

The advantage is clear: reduced human intervention when things break.  For
example, instead of having to page someone to bounce a pod, the operator can
instead bounce the pod itself.

Benefits of Operator
====================

**Increase operational consistency**

#. Improves ability to automate
#. Reduces learning curve; understanding one app means you understand all apps
#. Easily enforce operational standards from App SRE, Infosec, FedRAMP, etc.

**Simplify configuration for apps**

#. Most app teams do not have strong preferences for many operational aspects, e.g. logging
#. Reduce number and complexity of k8s resources app teams need to maintain
#. Reduces number of errors app teams can make (e.g. ServiceMonitor configuration)
#. Reduces onboarding ramp up time

**Adds ability to remediate common issues**

#. Bouncing pods
#. Running DB migration scripts
#. Autoscaling

**Add common alerts for apps**

#. Consumer group lag
#. Public service request rate
#. Public service error rate
#. Public service latency

**Facilitates deploying pre-production environments**

#. Abstracts various environment-specific operational components, e.g. RDS vs local DB
#. Dynamic consumer group config enables sharing of a singla Kafka instance
#. Facilitates deploying platform into dynamic namespace configuration
#. Significantly reduces the number of resources required to create a complete environment
#. Will be able to wire up metrics just like stage/prod
#. Will be able to set up web gateway similar to just like stage/prod

**Centralize operational evolution**

#. Abstract operational aspecs of apps, e.g. logging
#. Operational changes can then be applied platform-wide by changing the operator
#. Example: Introduction of Istio
#. Example: Migration of logging to kafka
#. Example: Modifying deployments or network policies for compliance requirements
#. Example: progressive deployment

Use Cases
=========

The operator should fulfill each of the following use cases, thus they should
strongly guide its design.

Personal Dev Environment
------------------------

Insights app developers use the operator to deploy personal instances of the
Insights Platform.  This will replace the use of Docker Compose or any other
orchestration tool used by developers to manage local environments.

Stage and Production
--------------------

The operator is used to deploy all applications in stage and production.  This
means the resources in an app's deployment template referenced by their saas
deploy file in app-interface will use CRDs managed by the Insights Operator
where applicable.  It also means that the operator must be stable enough to be
relied on for production usage.

Integration Tests in PR Check
-----------------------------

The PR check script uses the operator to deploy a temporary platform
environment to run tests.  Once the tests complete, the the PR check script
will remove the CRs, triggering the operator to tear down the temporary
environment.  Thus the operator needs to *quickly* set up and tear down
environments.  This should also be contained in one namespace, as opposed to
stage and production, where apps are spread across multiple namespaces.

Operational Remediations
------------------------

The operator should respond to typical issues that can be remediated
automatically.  One example would be overloaded pods, of which the solution
would be to increase the number of replicas for the affected deployment.

Custom Resource Definitions (CRDs)
==================================

The CRDs for an operator are considered the interface between an app developer
and the operator.  In other words, these CRDs will be how an app developer will
define what should be deployed in a given environment.  These CRDs will be used
to deploy the vast majority of resources in the platform.  App teams will be
expected to adopt these CRs in favor of lower level CRs like ``Deployment`` or
``Service``.

App teams will also be required to retrofit their applications to the standards
imposed by these CRs, e.g.  database and logging environment variable names,
pre-hook pod conventions, etc.

Application
-----------

This is intended to be a replacement for a single ``Deployment`` or
``DeploymentConfig``.  

This CR produces (or will produce):

* One ``Deployment`` resource.
* A ``Service`` resource for metrics. If a public web service is configured, it will add a
  port for the public endpoint.
* A ``ServiceMonitor`` resource to configure Prometheus to scrape the
  deployment's metrics endpoint.
* A ``PrometheusRule`` resource to create SLI/SLO alerts for the deployment.
* One or more ``KafkaTopics`` if the deployment intends to connect to Kafka.
* A routing configuration in the gateway if the deployment intends to run a
  publicly-exposed webservice.

Configuration:

* Image spec
* Public service name.  This will update the routing configuration
  for the nginx gateway (i.e. reverse proxy)
* Kafka topic names.
* Environment variables
* Liveness and readiness probes.  These could potentially be standardized.
* Resource constraints
* Replica scaling attributes

A couple of example CRs have been created for `advisor-api`_ (a public-facing web
service) and `advisor-service`_ (a kafka client).

.. _advisor-api: advisor/advisor-api.yml
.. _advisor-service: advisor/advisor-api.yml

Environment
------------

The ``Environment`` represents the foundation of an instance of
cloud.redhat.com.  The fact that any number of ``Environment`` resources can be
managed by a single operator means that the one operator can manage multiple
instances of a cloud.redhat.com deployment in parallel.

This CRD defines a set of core services to be deployed, including:

* Gateway router
* Entitlements service
* Kafka deployment
* UHC auth proxy
* Prometheus push gateway (only if existing push gateway config not provided)

Configuration:

* Auth/SSO configuration
* Database configuration (i.e. RDS or pod-based deployments)
* API prefix
* Logging config
* Prometheus config
* Prometheus push gateway config
* Entitlements service config
* publicly exposed?

``Application`` resources will always depend on one ``Environment``, referenced
by name in its ``base`` attribute.

``Environment`` will be defined in the same namespaces that the gateway is
deployed in.  Thus most, if not all, ``Applications`` will live in a different
namespace.

Changes to an ``Environment`` will propagate to all the associated
``Applications``.

Serivce Dependencies
====================

The operator will introduce the idea of service dependencies.  An ``Application``
can list out a number of service dependencies.  If any of the dependencies are
not met, then the operator will emit an event listing the missing service
dependencies and requeue the ``Application`` for reconciliation until the
dependencies are met.  This is similar to how Openshift already manages
dependencies, e.g. a missing persistent volume.

Service hostnames for dependent apps will be added to the JSON configuration
mounted into an app's container; hostnames for apps that are not listed as
dependencies will not be added to the configuration.

Gateway
=======

At this time the operator will intend to use the new gateway configuration base
on the `Turnpike`_ project.  This will enable the operator to dyanmically update
the routing configuration of the gateway as apps are deployed or removed.

A new CRD will be introduced to persist routing configuration:
``InsightsRoute``.  This resource will be automatically created by the
``Application`` controller, but devs can add these resources to their deployment
template if they are not using ``Application`` to deploy their public facing
service.

.. _Turnpike: https://github.com/RedHatInsights/turnpike

Application Conventions
=======================

In order for an app to be deployed via the Insights Operator, it must conform
to a number of conventions laid out by the operator.  If it does not, it will
either fail to deploy or the CR creation will be rejected due to validation
failures.

Configuration Volume
--------------------

Instead of adding a large number of environment variables to configure an app,
a JSON document will be compiled into a ``Secret`` resource and mounted into an
app's pod at ``/config``.  A non-exhaustive list of items to be configured in
this document:

* Kafka boostrap URL
* Other platform service URLs, e.g. RBAC and host inventory
* Web and metric port numbers
* Database hostname and credentials
* Available topics and the consumer group assigned to the app
* Cloudwatch/Logging configuration
* API prefix

Apps should be able to parse the JSON document and find configuration options
at consistent locations.  

Kafka
-----

Kafka topics will be listed in ``Applications``, and ``KafkaTopic`` resources
will be implicity created when an app references non-existent topics.  A topic
definition has an ``owner`` boolean field.  If it is set to ``true``, then the
CR can define the configuration of the topic, e.g. partition count or retention
time.  Only one app can be set as topic owner at a given time; apps that try to
claim ownership of a topic that is already owned will result in a rejection of
the app.  If an app requests a topic that has no owner, it will not be deployed
until the app that claims ownership is deployed.

Security
--------

* Apps will run with a service account with very few privileges.
* Apps may be able to request more specific privileges.
* App service accounts should never have any privileges outside its own namespace.

Convention Compliance
---------------------

* Kafka clients must consume provided consumer group and topic names
* Web services must consume provided API prefix and port number
* Apps must consume provided metrics path and port number

Most apps should be able to comfortably fit into the Application conventions.
If not, apps can specify their own pod template in their Application, but this
may cause an app to lose operational compliance.  Thus, use of a custom pod
template is discouraged unless absolutely necessary.

Common Library
==============

While apps can consume the JSON configuration directly, the platform team will
build and maintain a library that will expose an API to query the
configuration.  This API will contain both high level (e.g.
``get_django_db_config(db_name)``) and low level (e.g.
``get_db_password(db_name)``).

In addition, the library will include a logging package to abstract away the
log handler implementation.  This will allow the platform team to enforce
logging best practices while hopefully simplifying the burden of logging
configuration for apps.

The platform team will maintain configuration libraries for four languages:

* Python
* Ruby
* Java
* Go

These libraries are intended to:

#. Simplify configuration for app devs
#. Allow enforcement of compliance requirements and best practices

They are *not* intended to make configuration *more* difficult for app devs, nor
to be an extra burden for devs.

Operational Remediations
========================

The operator should be able to handle several well-known operations issues.
This will help reduce the number of indicents that require human intervention,
e.g. PagerDuty notifications.

Bounce a Stuck Pod
------------------

If a pod is considered "stuck" based on a Prometheus metric, e.g. request count
is less than one for ten minutes, then delete the pod and let the
``Deployment`` recreate.

Autoscaling
-----------

The operator will watch a number of metrics:

* For web applications, watch the gateway latency metric.
* For kafka applications, watch the topic lag.
* For any application, watch the average pod CPU percentage.

If any one of these metrics reaches a particular threshold, then increase the
number of replicas for the target ``Deployment``.  The reverse is also true:
Reduce the number of replicas (down to a minimum threshold) if it appears that
most pods are idle.

One Operator vs Many
====================

While each app team should be responsible for the operation of their own apps,
the cost of building and maintaining many operators significantly outweights
the benefit of placing greater operational responsibility on app teams.  Having
to create an operator -- even using the Operator SDK using a shared library
with examples -- is a high barrier to entry for any app team looking to build
an app on the Insights Platform.

In addition, the use of a shared library in multiple operators would introduce
versioning headaches as each app team would have to consume a constant stream
of library releases.

Splitting the Insights Operator into many per-app operators would make it more
difficult to monitor relationships between apps or platform components, e.g.
advisor's dependency on RBAC.  This makes it more difficult to self-heal when
these relationships break down, causing outages.

Thus the Platform team will take on the responsibility of maintaining the
Insights Operator.  The operator could eventually become the centerpiece of
operational value provided by the platform team; many other aspects of
platform-provided operational support will eventually be absorbed by the
operator.

Relationship to app-interface
=============================

Operator Responsibilities
-------------------------

While the operator could take over some of the responsibilities of
app-interface in the long-term, there are currently no plans to implement these
in the operator.  An example would be creating all requisite AWS resources from
the operator instead of tracking them in app-interface.

There should be little to no change in app-interface for the operator to be
utilized by apps; instead, an app's deployment template will be significanly
modified to push an ``Application`` instead a ``Deployment`` and ``Service``.
An app's ``ServiceMonitors`` and ``PrometheusRules`` may be removed from
app-interface if the operator starts creating these.  Namespaces, AWS
resources, secrets, and deployment pipelines are still managed via
app-interface.

App-interface will be queried to deploy pre-production environments, but this
will not be part of the operator; instead it should be decoupled from
app-interface except indirectly by looking for particular k8s resources laid
down by app-interface.

Operator Deployment
-------------------

The operator will be deployed via app-interface.  It is still to be determined
exactly how all of its components will be deployed.  Because this operator will
never be consumed directly by external customers, many the constraints placed on
shipped operators do not apply to this one.  That said, we will likely still
deploy this operator via OLM since that is how operators are required to be
deployed on OSD clusters managed by App SRE.

.. vim: tw=80
