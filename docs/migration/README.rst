Migrate to App-SRE Build Pipeline and Clowder
=============================================

Deployment and configuration of an app on cloud.redhat.com becomes much simpler
after migrating to Clowder because a lot of operational decisions are made for
the app, e.g. logging and kafka topic configuration.  In addition, migrating to
the operator unlocks the ability to leverage ephemeral environments for
smoke/integration testing.

The migration involves some work, of course:  apps must ensure conformity to the
conventions enforced by Clowder before they can be managed by it.

This migration combines two migrations into one: 

* Migrate build pipelines to app-interface
* Migrate apps to Clowder

Performing both migrations together reduces overall work, though you need to
perform more steps before seeing results.

Ensure code repo has a Dockerfile
---------------------------------

AppSRE's build conventions require that all images be built using a Dockerfile.
The Dockerfile can live anywhere in your code repo; you can configure a custom
location in your ``build_deploy.sh`` (described later) if it is placed somewhere
besides the root folder.

Note that a Dockerfile **must not** pull from Dockerhub.  AppSRE blocks all
requests to Dockerhub due to strict rate limiting imposed on their APIs.

Code changes to consume configuration
-------------------------------------

One of Clowder's key features is centralized configuration.  Instead of cobbling
together an app's configuration from a disparate set of secrets, environment
variables, and ``ConfigMaps`` that potentially change from environment to
environment, Clowder combines much of an app's configuration into a single JSON
document and mounts it in the app's container.  This insulates apps from
differences between environments, e.g. production, ephemeral, and local
development.

There is a companion client library for Clowder, currently implemented in `Go`_ and
`Python`_, that consumes the configuration document mounted into every application
container and exposes it via an API.  This API is the recommended way to consume
configuration that comes from Clowder.

Until a dev team is confident an app will not need to be deployed without
Clowder, please use an environment variable to switch between consuming
configuration from Clowder and from its current configuration method (e.g. env
vars, ``ConfigMap``).

Here are the items that you should consume from the Clowder client library:

* *Dependent service hostnames*: Look these up by the app name
* *Kafka bootstrap URL*: Multiple URLs can be provided, though only one is ever
  present today
* *Kafka topic names*: Please look up the actual topic name based on the requested
  name.
* *Web prefix and port number*
* *Metrics path and port number*

There are a couple of less trivial changes that may need to be made, depending
on what services are consumed by an app.

If object storage (e.g. S3) is used by an app, it is required that an app
switch to the `MinIO client library`_ if the app is intended to be deployed
outside of stage and production.  MinIO is deployed by Clowder in pre-production
environments as the object store provider, and the client library also supports
interacting with S3.  Thus switching to this library will allow an app to have
to use only one object storage library.

Clowder can provision Redis on behalf of an app.  If an app uses Redis, we
suggest testing with the version of Redis deployed by Clowder to ensure it is
compatible.  If not, changes to the app will need to be made.

.. _Go: https://github.com/RedHatInsights/app-common-go
.. _Python: https://github.com/RedHatInsights/app-common-python
.. _MinIO client library: https://github.com/minio/mc

Develop ``ClowdApp`` resource for target service
------------------------------------------------

An app's ``ClowdApp`` resourced will become its interface with Clowder.  It will
replace the app's ``Deployment`` and ``Service`` resources in its deployment
template.

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

Create deployment template with ``ClowdApp`` resource
-----------------------------------------------------

Going forward, an app's deployment template must live in its source code repo.
This will simply saas-deploy file configuration (see below) and has always been
AppSRE's convention.

Additional resources defined in an app's current deployment template besides
Deployment and Service should be copied over to the new template in the app's
source code repo.  Then the ``ClowdApp`` developed above should be added in.

A ``ClowdApp`` must point to a ``ClowdEnvironment`` resource via its ``envName`` spec
attribute, and its value should be set as the ``ENV_NAME`` template parameter.

Add ``build_deploy.sh`` and ``pr_check.sh`` to source code repo
---------------------------------------------------------------

AppSRE's build jobs largely rely on shell scripts in the target code repo to
execute the build and tests, respectively.  There are two jobs for each app:
"build master" and "PR check", and each job has a corresponding shell script:
``build_deploy.sh`` and ``pr_check.sh.``

``build_deploy.sh`` builds an app's image using a Dockerfile and pushes to Quay with
credentials provided in Jenkins job environment.  Make sure to push the ``latest``
and ``qa`` image tags if e2e-deploy backwards compatibility is needed.  There is
little variation in this file between projects, thus there are many examples to
pull from.

``pr_check.sh`` is where an app's unit test, static code analysis, linting, and
smoke/integration testing will be performed.  It is largely up to app owners
what goes into this script.  Smoke/integration testing will be performed by
bonfire, and there is an example script to paste into your app's script.  There
are a few environment variables to plug in at the top for an app, and the rest
of the script should be left untouched.

Both files live in the root folder of source code repo, unless overridden in the
Jenkins job definition (see below).

See examples of these files here:

https://github.com/RedHatInsights/insights-ingress-go/blob/master/build_deploy.sh

https://github.com/RedHatInsights/insights-ingress-go/blob/master/pr_check.sh


Create "PR check" and "build master" Jenkins jobs in app-interface
------------------------------------------------------------------

Two Jenkins jobs need to be defined for each app in app-interface: one to build
the image and one to run test validations against PRs.

AppSRE uses Jenkins Job Builder (JJB) to define jobs in YAML.  Jobs are created
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
your jobs defined.  AppSRE provides two Jenkins servers: ``ci-int`` for projects
hosted on gitlab.cee.redhat.com, and ``ci-ext`` for public projects hosted on
Github.  Note that private Github projects are **not supported**; if a Github
project must remain private, then its origin must move to gitlab.cee.redhat.com.

Create new saas-deploy file
---------------------------

The last step to enable smoke testing is to create a new saas-deploy file to
provide `Bonfire`_ with a way to deploy the app to an ephemeral environment. This saas file should
be separate from the existing saas file so that the template ``path`` can be different for each
service. It should not contain any of the ``stage`` or ``prod`` deployment targets until you are
ready to deploy the ``ClowdApp`` into those environments.

Points to ensure are in place in your new saas-deploy file:

* Add ``ClowdApp`` as a resource type
* Point ``resourceTemplate`` ``url`` and ``path`` to the deployment template in
  the app's code repo
* Remove ``IMAGE_TAG`` from the ``target``.  This was only specified because the
  deployment template was in a separate repo than the code.
* Add an ephemeral target.  This will be used by Bonfire to know how to deploy
  the app.  Example:

.. code-block:: yaml

    - namespace:
        $ref: /services/insights/ephemeral/namespaces/ephemeral-base.yml
      disable: true  # do not create an app-sre deploy job for ephemeral namespace
      ref: internal  # populated by bonfire
      parameters:
        REPLICAS: 1

Once these changes are merged into app-interface, you should be able to open a
PR against the app's source code repo and see Bonfire deploy the app, assuming
all dependent services are also set up with Bonfire.



Testing your migration code/template changes
-----------------------------------

**Testing Without Jenkins**


You can test the changes you are making to an application before any Jenkins jobs are wired up to
your app's repo. This allows you to test changes before they are merged into the app's repository
or app-interface. The only thing you *cannot* test is of course whether the Jenkins jobs work as
expected but you could even run ``pr_check.sh`` locally if you pass all the proper environment
variables to it.

#. Create a separate branch on the app repo and add:
    * the new deployment template defining your ``ClowdApp``
    * whatever code changes are needed so that your app can run as a ClowdApp
    * ``build_deploy.sh``
    * ``pr_check.sh``

#. Push your changes to git and note the git commit's hash.

#. Create a separate branch in app-interface and add:
    * an updated deploy.yaml/saas file that configures the ephemeral deployment target (make sure
      that the template path for the app points to your new ClowdApp template)

#. Clone `Bonfire`_ and install it

#. | Follow `these directions <https://github.com/RedHatInsights/bonfire#running-a-local-qontract-server>`_
    to set up your own local app-interface server and start it

#. Build your app's new docker image and somehow push it to quay with a unique image tag.
    * | You can use ``build_deploy.sh`` for this but be careful to not overwrite any tags like
        ``latest``, ``qa``, etc. You may want to push to an entirely separate quay repo just to be
        safe. But, you'll have to temporarily edit the ``IMAGE`` that your ``ClowdApp`` template uses
    * | If the app's quay repository is private or you do not have access rights to push to it, you
        may want to create your own temporary quay repo to do your testing, build the Dockerfile
        yourself, and ``docker push`` your built image to that repo. You will need to temporarily
        edit the application template to use your repo for the ``IMAGE``.

#. Log in to the ephemeral `cluster`_ using ``oc``.

#. | Run `the same deploy command <https://github.com/RedHatInsights/bonfire/blob/master/cicd/deploy_ephemeral_env.sh#L15-L20>`_
    that the ``pr_check.sh`` would run. This will reserve a namespace on the cluster and deploy your
    app into it. Make sure you replace the needed env vars: ``APP_NAME, COMPONENT_NAME, GIT_COMMIT,
    IMAGE, IMAGE_TAG``. The ``GIT_COMMIT`` should match the commit of your PR and the
    ``IMAGE/IMAGE_TAG`` should match whatever custom image you just built for the PR.

| 

.. _cluster: https://visual-app-interface.devshift.net/clusters#/openshift/crcs01ue1/cluster.yml

**Testing With Jenkins**


#. | Get all config changes related to the Jenkins jobs and saas file updates merged into
    app-interface so that the app has a pr_check job and an ephemeral deploy target in place. Your
    initial update to the saas file should only add the ephemeral deploy target, since you have not
    yet merged the new template into the app repo's ``master`` branch.

#. Open the PR against the app to add your ``build_deploy.sh``, ``pr_check.sh``, ``Dockerfile``,
    and new ClowdApp template.

#. The PR check test should fire and deploy the code changes you have made within your PR.


Disable builds in e2e-deploy
----------------------------

Once an app's build pipeline is set up through app-interface, the same build
pipeline in e2e-deploy/buildfactory needs to be disabled.  To do this, open a PR
against e2e-deploy that removes ``BuildConfig`` resources from the buildfactory
folder.  Remember to push the ``qa`` and ``latest`` tags from your
``build_deploy.sh`` script if you need backwards compatibility with e2e-deploy.

Note that in order to maintain compatibility with existing CI and QA
environments, the deployment templates for apps in e2e-deploy must be
maintained.

Deploy to stage and production
------------------------------

Once all the previous steps have been completed, it's time to deploy the
Clowder-dependent app to stage.  Move your ``target`` for stage to the new
saas-deploy file, ensuring ``ref`` is set to ``master``.  Note that this means
that all pushes to ``master`` will automatically be deployed to stage (per App
SRE convention).  Also remember to remove the ``IMAGE_TAG`` template parameter.

We should treat the deployment to stage as a test run for deploying to
production.  A cutover plan should account for the impact of an app's outage.
If the impact is low, the cutover plan can be simplified to save time and effort
in planning.  If the impact is high, then the cutover should be carefully
planned to ensure a little down time as possible.  If no additional care is
taken to minimize downtime, an app can expect 2-15 minutes of downtime, assuming
there are no regressions.

Once the app has been sufficiently validated in stage, follow the same process
to move the production target to the new saas-deploy file.  The only other
difference is that the ``ref`` for production should point to a git SHA.


.. _Bonfire: https://github.com/redhatinsights/bonfire 

.. vim: tw=80 spell spelllang=en
