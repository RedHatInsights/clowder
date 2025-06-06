# Migrate to App-SRE Build Pipeline and Clowder

Deployment and configuration of an app on cloud.redhat.com becomes much simpler after migrating to
Clowder because a lot of operational decisions are made for the app, e.g. logging and kafka topic
configuration. In addition, migrating to the operator unlocks the ability to leverage ephemeral
environments for smoke/integration testing.

The migration involves some work, of course: apps must ensure conformity to the conventions enforced
by Clowder before they can be managed by it.

This migration combines two migrations into one: 

* Migrate build pipelines to app-interface
* Migrate apps to Clowder

Performing both migrations together reduces overall work, though more steps need to be performed
before seeing results.

## Ensure code repo is visible

App SRE only supports fetching code without authentication.  This means that code must either be a
public repo on Github or on gitlab.cee.redhat.com.  If the app's code repo is currently private on
Github, it needs to either be made public or moved to Gitlab.  If the repo is already public on
Github or Gitlab, then the migration can proceed.

### Open Sourcing a Project

If it is decided to keep the project on Github, it must be open sourced.  This usually entails
several things:

* Adding a license file to the root of your repository
* Adding a reference to the license at the top of each source file
* Reviewing the repo's commit history for sensitive information

## Moving a Project from Github to Gitlab

Moving a project from Github to Gitlab should be relatively straightforward, but will require some
workflow changes for devs.

* Create a new repo on gitlab.cee.redhat.com. There is already an
  [insights-platform](https://gitlab.cee.redhat.com/insights-platform/)
  group that can be used to house repos, but other groups
  can be used if desired.
* Change your local upstream to gitlab.
  * Example: `git remote set-url upstream git@gitlab.cee.redhat.com:insights-platform/new-repo.git`

There should be no concern about existing CI integrations since the goal of this migration is to
have all CI integrated into app-interface.

### Why?

Jenkins jobs managed by App SRE do not support making authenticated pulls to git repos.

If a dev team is not comfortable making their code public to the world, then their interests would
be better served by moving to Gitlab since it is hosted within the Red Hat internal network, thus
reducing the risk of the code leaking publicly.

If a dev team *is* comfortable making their code public, then they should go ahead and schedule the
work to open source their app.  If this work is too costly, then their app should be hosted on
Gitlab until the open sourcing work can be completed.

Lastly, the Clouddot team's position has always been that private repositories live in Gitlab and
public ones live in Github.  By reserving Github repositories for open source projects, we can
better manage the number of Github licenses
required. 

## Ensure code repo has a Dockerfile

AppSRE's build conventions require that all images be built using a ``Dockerfile``. The
``Dockerfile`` can live anywhere in the code repo and can be configured by a custom location in
``build_deploy.sh`` (described later) if it is placed somewhere besides the root folder.

Note that a ``Dockerfile`` **must not** pull from Dockerhub.  AppSRE blocks all requests to
Dockerhub due to strict rate limiting imposed on their APIs.

## Code changes to consume configuration

One of Clowder's key features is centralized configuration.  Instead of cobbling together an app's
configuration from a disparate set of secrets, environment variables, and ``ConfigMaps`` that
potentially change from environment to environment, Clowder combines much of an app's configuration
into a single JSON document and mounts it in the app's container.  This insulates apps from
differences between environments, e.g. production, ephemeral, and local development.

There is a companion client library for Clowder, currently implemented in 
[Go](https://github.com/RedHatInsights/app-common-go),
[Python](https://github.com/RedHatInsights/app-common-python),
[Javascript](https://github.com/RedHatInsights/app-common-js) and 
[Ruby](https://github.com/RedHatInsights/app-common-ruby), that consumes the 
configuration document mounted into every application
container and exposes it via an API. 
This [API](https://github.com/RedHatInsights/clowder/blob/master/docs/appconfig/schema.md) 
is the recommended way to consume configuration that comes from Clowder.

Until a dev team is confident an app will not need to be deployed without Clowder, please use the
``isClowderEnabled`` method in the client library, or alternatively the presence of the
``ACG_CONFIG`` environment variable to switch between consuming configuration from Clowder and from
its current configuration method (e.g. env vars, ``ConfigMap``).

Here are the items that should be consumed from the Clowder client library:

* **Dependent service hostnames**: Look these up by the app name
* **Kafka bootstrap URL**: Multiple URLs can be provided, though only one is ever present today
* **Kafka topic names**: Please look up the actual topic name based on the requested name.
* **Kafka in the fedramp environment** requires the use of TLS + User authentication, both of which are supplied in the loaded config.
* **Web prefix and port number**
* **Metrics path and port number**

There are a couple of less trivial changes that may need to be made, depending on what services are
consumed by an app.

If object storage (e.g. S3) is used by an app, it is required that an app switch to the
[MinIO client library](https://github.com/minio/mc) if the app is intended to be deployed outside of
stage and production.  MinIO is deployed by Clowder in pre-production environments as the object
store provider, and the client library also supports interacting with S3.  Thus switching to this
library will allow an app to have to use only one object storage library.

Clowder can provision Redis on behalf of an app.  If an app uses Redis, the clouddot team suggests
testing with the version of Redis deployed by Clowder to ensure it is compatible.  If not, changes
to the app will need to be made.

## Develop ``ClowdApp`` resource for target service

An app's ``ClowdApp`` resource will become its interface with Clowder.  It will replace the app's
``Deployment`` and ``Service`` resources in its deployment template.

Developing the ``ClowdApp`` resource largely consists of two parts: 

* Copying over the relevant parts of an app's current pod template into the simplified pod spec
* Filling out the new metadata in the rest of the ``ClowdApp`` spec.

All deployments from one code repo should map to one ``ClowdApp``, each one mapping to an item in
the ``pods`` spec.  For each ``Deployment``, extract the following from the app's deployment
template in [saas-templates](https://gitlab.cee.redhat.com/insights-platform/saas-templates/):

* Image spec
* Resource requirements
* Command arguments
* Environment variables
* Liveness and readiness probes (not necessary if the app follows the ``/healthz`` convention)
* Volumes and volume mounts.

Additional information needed to fill out the other fields:

* List of kafka topics
* Optionally request a PostgreSQL database
* List of object store buckets
* Optionally request an in-memory database (i.e. Redis)
* List other app dependencies (e.g. ``rbac``)

The new ``ClowdApp`` can be validated on any cluster that has Clowder
installed. If access to a cluster with Clowder is not available, Clowder can be
[installed on Codeready Containers](https://github.com/RedHatInsights/clowder/blob/master/docs/crc-guide.md)
or on Minikube please see the [usage guide](../usage/getting-started.md) for more information.

## Create deployment template with ``ClowdApp`` resource

Going forward, an app's deployment template must live in its source code repo. This will simplify
saas-deploy file configuration (see below) and has always been AppSRE's convention.

Additional resources defined in an app's current deployment template besides Deployment and Service
should be copied over to the new template in the app's source code repo.  Then the ``ClowdApp``
developed above should be added in.

A ``ClowdApp`` must point to a ``ClowdEnvironment`` resource via its ``envName`` spec attribute, and
its value should be set as the ``ENV_NAME`` template parameter.

## Add ``build_deploy.sh`` and ``pr_check.sh`` to source code repo

AppSRE's build jobs largely rely on shell scripts in the target code repo to execute the build and
run tests, respectively.  There are two jobs for each app: *build master* and *PR check*, and each
job has a corresponding shell script: ``build_deploy.sh`` and ``pr_check.sh.``

``build_deploy.sh`` builds an app's image using a Dockerfile and pushes to Quay with credentials
provided in Jenkins job environment.  Make sure to push the ``latest`` and ``qa`` image tags if
e2e-deploy backwards compatibility is needed.  There is little variation in this file between
projects, thus there are many examples to pull from.

``pr_check.sh`` is where an app's unit test, static code analysis, linting, and smoke/integration
testing will be performed.  It is largely up to app owners as to the content of this script.
Smoke/integration testing will be performed by Bonfire, and there is an example script available to
paste into an app's script.  There are a few environment variables to plug in at the top for an app,
and the rest of the script should be left untouched.

Both files live in the root folder of source code repo, unless overridden in
the Jenkins job definition (see below).

See examples of these files here:

https://github.com/RedHatInsights/insights-ingress-go/blob/master/build_deploy.sh

https://github.com/RedHatInsights/insights-ingress-go/blob/master/pr_check.sh


## Create "PR check" and "build master" Jenkins jobs in app-interface

Two Jenkins jobs need to be defined for each app in app-interface: one to build the image and one to
run test validations against PRs.

AppSRE uses Jenkins Job Builder (JJB) to define jobs in YAML.  Jobs are created by referencing job
templates and filling in template parameters.  There are two common patterns: one for github repos
and another for gitlab repos.

Github:
```yaml
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
```

Gitlab:
```yaml
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
```

In the app's ``build.yml``, a specification must be made as to which Jenkins server to have the apps
jobs defined.  AppSRE provides two Jenkins servers: ``ci-int`` for projects hosted on
gitlab.cee.redhat.com, and ``ci-ext`` for public projects hosted on Github.  Note that private
Github projects are **not supported**; if a Github project must remain private, then its origin must
move to gitlab.cee.redhat.com.

## Create new saas-deploy file

The last step to enable smoke testing is to create a new saas-deploy file to provide
[Bonfire](https://github.com/redhatinsights/bonfire) with a way to deploy the app to an ephemeral
environment. This saas file should be separate from the existing saas file so that the template
``path`` can be different for each service. It should not contain any of the ``stage`` or ``prod``
deployment targets until you are ready to deploy the ``ClowdApp`` into those environments.

Points to ensure are in place in your new saas-deploy file:

* Add ``ClowdApp`` as a resource type
* Point ``resourceTemplate`` ``url`` and ``path`` to the deployment template in
  the app's code repo
* Remove ``IMAGE_TAG`` from the ``target``.  This was only specified because the
  deployment template was in a separate repo than the code.
* Add an ephemeral target.  This will be used by Bonfire to know how to deploy
  the app. Example:

```yaml
- namespace:
    $ref: /services/insights/ephemeral/namespaces/ephemeral-base.yml
  disable: true  # do not create an app-sre deploy job for ephemeral namespace
  ref: internal  # populated by bonfire
  parameters:
    REPLICAS: 1
```

Once these changes are merged into app-interface, it should be possible to open a PR against the
app's source code repo and see Bonfire deploy the app, assuming all dependent services are also set
up with Bonfire.

## Testing your migration code/template changes

### Testing Without Jenkins

Changes to an application may be tested before any Jenkins jobs are wired up to the app's repo. This
allows the testing of changes before they are merged into the app's repository or app-interface. The
only thing that *cannot* be tested is of course whether the Jenkins jobs work as expected but it is
even possible to run ``pr_check.sh`` locally if all the proper environment variables are passed to
it.

* Create a separate branch on the app repo and add:
  * the new deployment template defining your ``ClowdApp``
  * whatever code changes are needed so that your app can run as a ClowdApp
  * ``build_deploy.sh``
  * ``pr_check.sh``
* Push your changes to git and note the git commit's hash.
* Create a separate branch in app-interface and add:
  * an updated deploy.yaml/saas file that configures the ephemeral deployment target (make sure
  that the template path for the app points to the new ClowdApp template)

* Clone [Bonfire](https://github.com/redhatinsights/bonfire) and install it

* Follow [these directions](https://github.com/RedHatInsights/bonfire#running-a-local-qontract-server)
  to set up a local app-interface server and start it

* Build the app's new docker image and somehow push it to quay with a unique image tag.
  * You can use ``build_deploy.sh`` for this but be careful to not overwrite any tags like
  ``latest``, ``qa``, etc. It may be better to push to an entirely separate quay repo just to be
  safe. But, you'll have to temporarily edit the ``IMAGE`` that your ``ClowdApp`` template uses
  * If the app's quay repository is private or access rights are not available, a temporary quay repo
  could be created to perform testing. Build the Dockerfile
  and ``docker push`` the built image to that repo. The application template will need editing
  temporarily to use the temporary repo for the ``IMAGE``.

* Log in to the ephemeral [cluster](https://visual-app-interface.devshift.net/clusters#/openshift/insights/c-rh-c-eph/cluster.yml) using ``oc``.

* Run [the same deploy command](https://github.com/RedHatInsights/bonfire/blob/master/cicd/deploy_ephemeral_env.sh#L15-L20)
that the ``pr_check.sh`` would run. This will reserve a namespace on the cluster and deploy your
app into it. Make sure you replace the needed env vars: ``APP_NAME``, ``COMPONENT_NAME``, ``GIT_COMMIT``,
``IMAGE``, ``IMAGE_TAG``. The ``GIT_COMMIT`` should match the commit of the PR and the
``IMAGE/IMAGE_TAG`` should match whatever custom image was just built for the PR.


### Testing With Jenkins

* Get all config changes related to the Jenkins jobs and saas file updates merged into
app-interface so that the app has a pr_check job and an ephemeral deploy target in place. The
initial update to the saas file should only add the ephemeral deploy target, since the new
template has not yet been merged into the new template into the app repo's ``master`` branch.
* Open the PR against the app to add the ``build_deploy.sh``, ``pr_check.sh``, ``Dockerfile``,
and new ClowdApp template.

* The PR check test should fire and deploy the code changes that have been made within the PR.


## Disable builds in e2e-deploy

Once an app's build pipeline is set up through app-interface, the same build pipeline in
e2e-deploy/buildfactory needs to be disabled.  To do this, open a PR against e2e-deploy that removes
``BuildConfig`` resources from the buildfactory folder.  Remember to push the ``qa`` and ``latest``
tags from the ``build_deploy.sh`` script if backwards compatibility is needed with e2e-deploy.

Note that in order to maintain compatibility with existing CI and QA environments, the deployment
templates for apps in e2e-deploy must be maintained.

## Deploy to stage and production

Once all the previous steps have been completed, it's time to deploy the Clowder-dependent app to
stage.  Move the ``target`` for stage to the new saas-deploy file, ensuring ``ref`` is set to
``master``.  Note that this means that all pushes to ``master`` will automatically be deployed to
stage (per App SRE convention).  Also remember to remove the ``IMAGE_TAG`` template parameter.

The deployment to stage shoudl be treated as a test run for deploying to production.  A cutover plan
should account for the impact of an app's outage. If the impact is low, the cutover plan can be
simplified to save time and effort in planning.  If the impact is high, then the cutover should be
carefully planned to ensure a little down time as possible.  If no additional care is taken to
minimize downtime, an app can expect 2-15 minutes of downtime, assuming there are no regressions.

Once the app has been sufficiently validated in stage, follow the same process to move the
production target to the new saas-deploy file.  The only other difference is that the ``ref`` for
production should point to a git SHA.
