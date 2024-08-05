# Clowder Readme
## What is Clowder?

Clowder is a kubernetes operator designed to make it easy to deploy applications
running on the cloud.redhat.com platform in production, testing and local
development environments.

[Learn More](learn-more)

## See Clowder in Action

![Animated GIF terminal example](img/terminal-example.gif)

## Why use Clowder?

In addition to reducing the effort to maintain a deployment template, Clowder
provides a number of other benefits:

* **Consistent deployment** Whether you're deploying to production, running smoke
  tests on a PR, or developing your application locally, Clowder will use the
  same app definition for all three. No more endlessly tweaking environment variables!
* **Focus on development** Clowder has the best practices of running an app in
  a microservices environment as well as specific requirements from the app-sre
  team, such as pod affinity, rollout parameters etc built-in. Spend less time
  worrying about configuring deployment templates and more time writing your app.
* **Assisting Ops** Any dev or SRE that learns how Clowder deploys apps will
  implicitly understand the deployment of any other app utilizing Clowder.
* **Deploy a full environment locally** Gone are the days of hacking together
  scripts that just about get you mocked or partially working dependant services.
  With  Clowder, you can deploy an instance of the cloud.redhat.com platform on your
  local laptop, or in a dev cluster to use as you wish.

Clowder will provision resources depending on the mode choosen for each provider,
and will return a consistently formatted JSON configuration document for each app
to consume, leaving teams to focus more on writing code than differences between
environments. The Clowder config client can assist with this and currently has support
for Python, Go, Javascript and Ruby.

![Configuration model](img/config.svg)

## Feature List

Clowder currently features support for:

* Kafka Topics
* Object Storage
* PostgreSQL Database
* In-Memory DB
* Feature Flags (development only)
* CronJob support
* Jobs Support

## Roadmap

Our current roadmap looks like this:

* Autoscaling (possibly via [Keda](https://github.com/kedacore/keda))
* Dynamic routing for public web sevices
* Automatic metrics configuration
* Automatic network policy configuration
* Standard, configurable alerting: Error rate, latency, Kafka topic lag, etc
* Canary deployments (possibly via [Flagger](https://github.com/weaveworks/flagger))
* Operational remediations
* Observe service status from a `ClowdApp` `status`, based on sevice dependencies.
* Istio integration
* Advanced logging configurations (e.g. logging messages to kafka topics)

## Getting Clowder

**Clowder is already running in pre-prod/prod environments.**

To run Clowder locally in Minikube, obtain and install
[Minikube](https://minikube.sigs.k8s.io/docs/start/).

Clowder is developed on Fedora and the kvm driver has been found to work best
initiated with the following options:

```shell
minikube start --cpus 4 --disk-size 36GB --memory 16000MB --driver=kvm2 --addons registry --addons ingress  --addons=metrics-server --disable-optimizations
```

NOTE:
Mac OS is also supported with the `virtualbox` and `hyperkit` drivers. A full
guide [can be found here](./macos)

To persist these changes for every minikube invocation, run the following:

```shell
minikube config set cpus 4
minikube config set memory 16000
minikube config set disk-size 36GB
minikube config set driver kvm2
```

If you encounter any kvm issues, please take a look
[at the troubleshooting guide](./developer-guide)

The ``kube_setup.sh`` script then needs to be run by invoking

```shell
./build/kube_setup.sh
```

Clowder can then be installed by running:

```shell
# Be sure to get the latest release in the link above!
minikube kubectl -- apply -f $(curl https://api.github.com/repos/RedHatInsights/clowder/releases/latest | jq '.assets[0].browser_download_url' -r) --validate=false
```

## Usage

To use Clowder to deploy an application a ``ClowdEnvironment`` resource must be
present to define an environment. Once this has been deployed, a ``ClowdApp``
resource is authored for the app and deployed alongside the ``ClowdEnvironment``.

Example app developer workflow:

* Install Clowder on a minikube environment.
* Use ``kubectl apply -f clowdenv.yaml`` to apply a ``ClowdEnvironment`` resource
  to the cluster.
* Use ``kubectl apply -f clowdapp.yaml`` to apply a ``ClowdApp`` resource to the
  cluster.

More details on how to do this are present in the [Getting Started](usage/getting-started) section
of the documentation.

[API Reference](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html)

## Building Clowder

If you want to run a version of Clowder other than the released version there
are a few prerequisites you will need. To learn about developing Clowder please
visit the xref:docs/antora/modules/ROOT/pages/developer-guide.adoc[developing clowder] page for more detailed instructions.

## History

To understand more about the design decisions made while developing Clowder,
please visit the xref:docs/antora/modules/ROOT/pages/clowder-design.adoc[design document]

## Connect

Any questions, please ask one of the Clowder development team

* [@psav](https://github.com/psav)
* [@bsquizz](https://github.com/bsquizz)
* [@bennyturns](https://github.com/bennyturns)
* [@adamrdrew](https://github.com/adamrdrew)
