Migrate to App-SRE Build Pipeline and Clowder
=============================================

Deployment and configuration of an app on cloud.redhat.com becomes much simpler
after migrating to Clowder because a lot of operational decisions are made for
the app, e.g. logging and kafka topic configuration. The migration involves some
work, of course:  apps must ensure conformity to the conventions enforced by
Clowder before they can be managed by it.

This migration combines two migrations into one: 

* Migrate build pipelines to app-interface
* Migrate apps to Clowder

Performing both migrations together reduces overall work, though you need to
perform more steps before seeing results.

Ensure code repo has a Dockerfile
---------------------------------

App SRE's build conventions require that all images be built using a Dockerfile.  
The Dockerfile can live anywhere in your code repo; you can configure a custom
location in your build_deploy.sh (described later) if you place it somewhere
besides the root folder.

Note that a Dockerfile must not pull from Dockerhub.  App SRE blocks all
requests to Dockerhub due to strict rate limiting imposed on their APIs.

Code changes to consume configuration
-------------------------------------

One of Clowder's key features is centralized configuration.  Instead of cobbling
together an app's configuration from a disparate set of secrets, environment
variables, and ConfigMaps that potentially change from environment to
environment, Clowder combines much of an app's configuration into a single JSON
document and mounts it in the app's container.  This should also insulate apps
from differences between environments, e.g. production, ephemeral, and local
development.

There is a companion client library for Clowder, currently implemented in Go and
Python, that consumes the configuration document mounted into every application
container and exposes it via an API.  This API is the recommended way to consume
configuration that comes from Clowder.

Until a dev team is confident an app will not need to be deployed without
Clowder, please use an environment variable to switch between consuming
configuration from Clowder and from its current configuration method (e.g. env
vars, ConfigMap).

Here are the items that you should consume from the Clowder client library:

* Dependent service hostnames: Look these up by the app name
* Kafka bootstrap URL: Multiple URLs can be provided, though only one is ever
  present today
* Kafka topic names: Please look up the actual topic name based on the requested
  name.
* Web prefix and port number
* Metrics path and port number

There are a couple of less trivial changes that may need to be made, depending
on what services are consumed by an app.

If object storage, i.e. S3, is used by an app, it is recommended that an app
switch to the MinIO client library.  MinIO is used in pre-production
environments, and it also supports interacting with S3.  Thus switching to this
library will allow an app to have to include only one object storage client
library.

Clowder can provision Redis on behalf of an app.  If an app uses Redis, we
suggest testing with the version of Redis deployed by Clowder to ensure it is
compatible.  If not, changes to the app will need to be made.

Develop ClowdApp resource for target service
--------------------------------------------

Developing the ``ClowdApp`` resource largely consists of two parts: 

* Copying over the relevant parts of an app's current pod template into the
  simplified pod spec
* Filling out the new metadata in the rest of the ``ClowdApp`` spec.

All deployments from one code repo should map to one ``ClowdApp``, each one
mapping to an item in the ``pods`` spec.  For each ``Deployment``, extract the
following from the app's deployment template in `saas-templates`_:

* image spec
* resource requirements
* command arguments
* environment variables
* liveness and readiness probes
* volumes and volume mounts.

Additional information needed to fill out the other fields:

* List of kafka topics
* Optionally request a PostgreSQL database
* List of object store buckets
* Optionally request an in-memory database (i.e. Redis)
* List other app dependencies (e.g. ``rbac``)

The new ``ClowdApp`` can be validated on any cluster that has Clowder installed.
If access to a cluster with Clowder is not available, Clowder can be `installed
on Codeready Containers`_.

.. _example: https://github.com/RedHatInsights/insights-puptoo/blob/fea32bef660802b0647f616bc211fb52f24a30e5/deployment.yaml
.. _saas-templates: https://gitlab.cee.redhat.com/insights-platform/saas-templates/
.. _installed on Codeready Containers: https://github.com/RedHatInsights/clowder/blob/master/docs/crc-guide.md

Add build_deploy.sh and pr_check.sh to source code repo
-------------------------------------------------------

App SRE's build jobs largely rely on shell scripts in the target code repo to
execute the build and tests, respectively.  There are two jobs for each app:
"build master" and "PR check", and each job has a corresponding shell script:
build_deploy.sh and pr_check.sh.

build_deploy.sh builds an app's image using a Dockerfile and pushes to quay with
credentials provided in Jenkins job environment.  Make sure to push the latest
and qa image tags if e2e-deploy backwards compatibility is needed.  There is
little variation in this file between projects, thus there are many examples to
pull from.

pr_check.sh is where an app's unit test, static code analysis, linting, and
smoke/integration testing will be performed.  It is largely up to app owners
what goes into this script.  Smoke/integration testing will be performed by
bonfire, and there is an example script to paste into your app's script.  There
are a few environment variables to plug in at the top for an app, and the rest
of the script should be left untouched.

Both files live in the root folder of source code repo, unless overridden in the
Jenkins job definition (see below).

Create "PR check" and "build master" Jenkins jobs in app-interface
------------------------------------------------------------------

Two Jenkins jobs need to be defined for each app in app-interface: one to build
the image and one to run test validations against PRs.

App SRE uses Jenkins Job Builder (JJB) to define jobs in YAML.  Jobs are created
by referencing job templates and filling in template parameters.  There are two
common patterns: one for github repos and another for gitlab repos.

Github:

.. code-block:: yaml

    project:
      name: puptoo-stage
      label: insights
      node: insights
      gh_org: RedHatInsights
      gh_repo: insights-puptoo
      quay_org: cloudservices
      jobs:
      - "insights-gh-pr-check":
          display_name: puptoo pr-check
      - "insights-gh-build-master":
          display_name: puptoo build-master

Gitlab:

.. code-block:: yaml

    project:
      name: insightsapp-poc-ci
      label: insights
      node: insights
      gl_group: bsquizza
      gl_project: insights-ingress-go
      quay_org: cloudservices
      jobs:
      - 'insights-gl-pr-check':
          display_name: 'insightsapp-poc pr-check'
      - 'insights-gl-build-master':
          display_name: 'insightsapp-poc build-master'


In your app's build.yml, you need to specify on which Jenkins server to have
your jobs defined.  App SRE provides two Jenkins servers: ci-int for projects
hosted on gitlab.cee.redhat.com, and ci-ext for public projects hosted on
Github.  Note that private Github projects are **not supported**; if a Github
project must remain private, then its origin must move to gitlab.cee.redhat.com.

Disable builds in e2e-deploy
----------------------------

Once an app's build pipeline is set up through app-interface, the same build
pipeline in e2e-deploy/buildfactory needs to be disabled.  To do this, open a PR
against e2e-deploy that removes ``BuildConfig`` resources from the buildfactory
folder.  Remember to push the ``qa`` and ``latest`` tags from your
``build_deploy.sh`` script if you need backwards compatibility with e2e-deploy.

Create deployment template with ClowdApp resource
-------------------------------------------------

Going forward, an app's deployment template must live in its source code repo.
This will simply saas-deploy file configuration (see below) and has always been
App SRE's convention.

Additional resources defined in an app's current deployment template besides
Deployment and Service should be copied over to the new template in the app's
source code repo.  Then the ClowdApp developed above should be added in.

A ClowdApp must point to a ClowdEnvironment resource via its ``envName`` spec
attribute, and its value should be set as the ``ENV_NAME`` template parameter.

Modify saas-deploy file for service
-----------------------------------

Once all the previous steps have been completed, it's time to deploy the
Clowder-dependent app.  This can be a little tricky since we need to keep the
current production app running smoothly until we are confident the new changes
are ready to be promoted.

It's likely best to create a separate saas-deploy file for Clowder-based
services.  The current ``deploy.yml`` can be cloned, with the name appended with
``-clowder`` and all the ``resourceTemplates`` removed.  Apps on Github need to
refer to ``ci-ext`` Jenkins, while apps on Gitlab should reference ``ci-int``.
If there are multiple services in one saas-deploy file which are not hosted by
the same provider, then two separate saas-deploy files must be maintained since
a saas-deploy file can only point to one Jenkins instance.

Points to ensure are in place in your new saas-deploy file:

* Add ``ClowdApp`` as a resource type
* Point ``resourceTemplate`` ``url`` and ``path`` to the deployment template in
  the app's code repo
* Remove ``IMAGE_TAG`` from all targets.  This was only specified because the
  deployment template was in a separate repo than the code.
* Ensure ref is set to master for stage and a git SHA for production.  Note that
  this means that all pushes to ``master`` will automatically be deployed to
  stage (per App SRE convention).
* Add an ephemeral target.  This will be used by Bonfire to know how to deploy
  the app.  Example:

.. code-block:: yaml

    - namespace:
        $ref: /services/insights/ephemeral/namespaces/ephemeral-base.yml
      disable: true  # do not create an app-sre deploy job for ephemeral namespace
      ref: internal  # populated by bonfire
      parameters:
        REPLICAS: 1

.. vim: tw=80 spell spelllang=en
