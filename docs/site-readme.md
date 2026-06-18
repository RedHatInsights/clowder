![Clowder - Clowd Platform Operator][clowder-logo]

![Build Passing][build-badge]
![Downloads][downloads-badge]
![Release][release-badge]
![Go Report Card][goreport-badge]

## What is Clowder?

Clowder is a Kubernetes operator designed to make it easy to deploy applications
running on the cloud.redhat.com platform in production, testing and local
development environments.

Learn more about Clowder in the [extended overview][learn-more].

## See Clowder in Action

![Animated GIF terminal example][terminal-example]

## Why use Clowder?

In addition to reducing the effort to maintain a deployment template, Clowder
provides a number of other benefits:

- **Consistent deployment** Whether you're deploying to production, running smoke
  tests on a PR, or developing your application locally, Clowder will use the
  same app definition for all three. No more endlessly tweaking environment variables!
- **Focus on development** Clowder has the best practices of running an app in
  a microservices environment as well as specific requirements from the app-sre
  team, such as pod affinity, rollout parameters etc built-in. Spend less time
  worrying about configuring deployment templates and more time writing your app.
- **Assisting Ops** Any dev or SRE that learns how Clowder deploys apps will
  implicitly understand the deployment of any other app utilizing Clowder.
- **Deploy a full environment locally** Gone are the days of hacking together
  scripts that just about get you mocked or partially working dependant services.
  With Clowder, you can deploy an instance of the cloud.redhat.com platform on your
  local laptop, or in a dev cluster to use as you wish.

Clowder provisions resources depending on the mode chosen for each provider and
returns a consistently formatted JSON configuration document (`cdappconfig.json`)
for each app to consume. The Clowder config client currently supports Python,
Go, JavaScript, and Ruby.

![Configuration model][config-model]

## Feature List

Clowder currently supports:

- Kafka topics
- Object storage
- PostgreSQL database
- In-memory DB (Redis)
- Feature flags (development only)
- CronJob support
- Job support

## Prerequisites

- Go 1.25
- `kubectl` configured against a target cluster or Minikube
- `make`
- Podman or Docker (for container image builds)

## Getting Clowder

**Clowder is already running in pre-prod/prod environments.**

To run Clowder locally in Minikube, obtain and install [Minikube][minikube].

Clowder is developed on Fedora and the kvm driver has been found to work best
with the following options:

```shell
minikube start --cpus 4 --disk-size 36GB --memory 16000MB --driver=kvm2 \
  --addons registry --addons ingress --addons=metrics-server \
  --disable-optimizations
```

To persist these settings for every Minikube invocation:

```shell
minikube config set cpus 4
minikube config set memory 16000
minikube config set disk-size 36GB
minikube config set driver kvm2
```

Then run the cluster setup script:

```shell
./build/kube_setup.sh
```

Install the latest Clowder release:

```shell
minikube kubectl -- apply -f \
  $(curl https://api.github.com/repos/RedHatInsights/clowder/releases/latest \
    | jq '.assets[0].browser_download_url' -r) \
  --validate=false
```

macOS is also supported. See the [macOS guide][macos-guide].

If you encounter kvm issues, see the [developer guide][developer-guide].

## Usage

To deploy an application with Clowder, a `ClowdEnvironment` resource must be
present to define the environment. Once deployed, author a `ClowdApp` resource
for the app:

```shell
kubectl apply -f clowdenv.yaml
kubectl apply -f clowdapp.yaml
```

Example manifests are in `docs/examples/`. Full instructions are in the
[Getting Started guide][getting-started].

See the [API reference][api-reference] for all resource fields.

## Building Clowder

Build the manager binary:

```shell
make build
```

Build the container image:

```shell
make docker-build
```

## Development Setup

Install build tool dependencies (controller-gen, kustomize, setup-envtest) via
the Go tools directive:

```shell
make update-deps
```

### Code generation

Run these commands after modifying the corresponding source:

| Change | Command |
| --------------------------------- | -------------------- |
| CRD types in `apis/` | `make generate && make manifests` |
| JSON schema in `schema/` | `make genconfig` |
| RBAC markers in controllers | `make manifests` |

### Running tests

Unit tests (requires envtest):

```shell
make test
```

End-to-end KUTTL tests (requires a running cluster):

```shell
make deploy-minikube-quick   # deploy latest build first
make kuttl
```

Run a single KUTTL test:

```shell
make kuttl KUTTL_TEST="--test=test-basic-app"
```

### Code quality

```shell
make fmt      # format with gofmt
make vet      # run go vet
make lint     # run golangci-lint
make pre-push # run all pre-commit checks
```

### Local development

```shell
make install           # install CRDs into cluster
make run               # run controller locally against kubeconfig cluster
make deploy-minikube   # build and deploy to minikube
```

For detailed instructions including debugging with Delve and VS Code, see the
[developer guide][developer-guide].

## Architecture

Clowder uses a provider-based plugin architecture. Each service type
(database, Kafka, object storage, etc.) is implemented as a pluggable provider
that can be swapped per environment via `ClowdEnvironment` configuration.

For internal design decisions, the resource cache, watch/filter system, and
code generation pipeline, see [ARCHITECTURE.md][architecture].

## Contributing

Contribution guidelines, commit conventions (Conventional Commits), pull request
flow, and testing patterns are described in [CONTRIBUTING.md][contributing].

## History

To understand more about the design decisions made while developing Clowder,
see the [design document][clowder-design].

## Connect

Questions? Reach out to the Clowder development team:

- [@bsquizz][bsquizz]
- [@bennyturns][bennyturns]
- [@adamrdrew][adamrdrew]
- [@maknop][maknop]

## License

Licensed under the [Apache License 2.0][license].

[clowder-logo]: docs/img/clowder.svg
[build-badge]: https://img.shields.io/github/actions/workflow/status/RedHatInsights/clowder/package.yml?branch=master
[downloads-badge]: https://img.shields.io/github/downloads/RedHatInsights/clowder/total.svg
[release-badge]: https://img.shields.io/github/v/release/RedHatInsights/clowder
[goreport-badge]: https://goreportcard.com/badge/github.com/RedHatInsights/clowder
[terminal-example]: docs/img/terminal-example.gif
[config-model]: docs/img/config.svg
[learn-more]: docs/learn-more.md
[minikube]: https://minikube.sigs.k8s.io/docs/start/
[macos-guide]: docs/macos.md
[developer-guide]: docs/developer-guide.md
[getting-started]: docs/usage/getting-started.md
[api-reference]: https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html
[architecture]: ARCHITECTURE.md
[contributing]: CONTRIBUTING.md
[clowder-design]: docs/clowder-design.adoc
[bsquizz]: https://github.com/bsquizz
[bennyturns]: https://github.com/bennyturns
[adamrdrew]: https://github.com/adamrdrew
[maknop]: https://github.com/maknop
[license]: LICENSE
