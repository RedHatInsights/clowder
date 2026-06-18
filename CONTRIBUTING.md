# Developing for Clowder

## Providers

### Provider setup

The core of Clowder revolves around a core concept of providers. Clowder splits its functionality
into multiple units called providers. These are called in a defined order and provide both
environmental and application level resources, as well as configuration that ultimately lands in the
`cdappconfig.json`.

The providers live in the `controllers/cloud.redhat.com/providers` folder and comprise of a 
`provider.go` file, some mode files, some implementation files and potentially some tests too.

The `provider.go` defines several key pieces. Shown below is the `deployment` provider's
`provider.go` file:

```golang
package deployment

import (
	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	apps "k8s.io/api/apps/v1"
)

// ProvName sets the provider name identifier
var ProvName = "deployment"

// CoreDeployment is the deployment for the apps deployments.
var CoreDeployment = rc.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})

// GetEnd returns the correct end provider.
func GetDeployment(c *p.Provider) (p.ClowderProvider, error) {
	return NewDeploymentProvider(c)
}

func init() {
	p.ProvidersRegistration.Register(GetDeployment, 0, ProvName)
}
```

The `ProvName` is an identifier that defines the name of the provider. Notice that the Golang
package name is the same as this identifier. This is a nice convention and one which should be
maintained when new providers are added. The next declaration is a MultiResourceIdent. These will be
discussed in a little more detail below, but in short, this is a declaration of the resources that
this particular provider will create.

After that, there is the `GetDeployment()` function. Every provider has some kind of `Get*()`
function, which is responsible for creating deciding which mode to run the provider in. Depending on
the environmental settings, providers can be run in different modes. The `deployment` provider is
a core provider and as such as no modal configuration, i.e. there is only one mode. Providers with
no modes will use the `default.go` to provide their functionality. The `Get*()` function returns
a Provider object. In this case the function is called `NewDeploymentProvider()` and returns the
default `DeploymentProvider` object. This will be expanded upon shortly.

The `init()` call is responsible for registering this provider with the internal provider
registration system. The provider's `Get*()` function is passed in, as well as an integer and the
`ProvName`. The integer specifies the order in which the providers should be run. `0` is the
first provider and `99`, by convention, is the last. Two providers can share the same order
number.

Care must be taken when providers depend on each others resources, that they are executed in the
correct order, otherwise the dependant provider will fail when its dependency is missing from the
cache. This will be explained in more depth in the caching section later in this document.

### Provider functionality

The `default.go` file is shown below:

```golang
package deployment

import (
    crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
    "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
    p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
)

type deploymentProvider struct {
    p.Provider
}

func NewDeploymentProvider(p *p.Provider) (p.ClowderProvider, error) {
    return &deploymentProvider{Provider: *p}, nil
}

func (dp *deploymentProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

    for _, deployment := range app.Spec.Deployments {

        if err := dp.makeDeployment(deployment, app); err != nil {
            return err
        }
    }
    return nil
}
```

The `default.go` file defines a singular mode for a provider. In other providers there may be
several modes and each of these will be housed in its own `.go` file, though it will be a part of
the same package. The `deploymentProvider` struct defines an struct to which functions are
attached for provider operation. Some of these can be internal, but the most important one is called
`Provide` and must be exported.

When the providers are _invoked_ in Clowder, they are done so in the two controllers,
`ClowdEnvironment` and `ClowdApp`. The `ClowdEnvironment` controller only runs the
_environmental_ functionality to provider environmental resources. An example of this would be a
kafka or obejct storage server, as there is only ever one of these per environment. The
`NewDeploymentProvider()` function, as referenced in the previous `provider.go` file, is
responsible for creating and managing these _environment_ level resources. These are run by the
_environment_ controller and will be reconciled whenever the `ClowdEnvironment` is triggered.

By contrast, `ClowdApp` modifications trigger the _application_ reconciliation, which first runs
the _environment_ function, in this case `NewDeploymentProvider()` before then running the
`Provide()` function. This may seem odd and indeed is a design quirk of Clowder that will
hopefully be resolved in a later release. Its reasoning is that the environmental resources often
need to provide information to the application level reconciliation, for instance to furnish the
`cdappconfig` with the Kafka broker address. Since this information is calculated by the
environment controller, the application controller must first rerun the environment controller's
functions to obtain the information.

Environment and application level functions can access and edit the `AppConfig` object which will
ultimately be transformed into the `cdappconfig.json` file that ends up in the app container at
runtime.

### Caching resources

A key tenet of the Clowder provider system is that of sharing resources. Without resource sharing,
providers that need to modify the resources of other providers result not only in multiple calls to
update the same resources, but also can potentially trigger multiple reconciliations as updates to
Clowder owned resources can trigger these.

To reduce this burden, the Clowder system will only apply resources at the very end of the
reconciliation. Until that time, resources are stored in the resource cache and providers are able
to retrieve objects from this cache, update them, and then placed the updated versions back in the
cache, so that their changes will be applied at the end of the reconciliation. This is where the
order of provider invocation is important.

The following is a snippet from the `deployment` provider's `provider.go`:

```golang
// CoreDeployment is the deployment for the apps deployments.
var CoreDeployment = p.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})
```

This was shown previously and is responsible for creating an object that can identify certain
resources. The call takes three arguments: the provider name, a purpose string (which details
briefly what the resource is used for), and a _template_ object.

NOTE: The template object is never *used* in anyway. It is merely there to determine the type of the resource.

In the `impl.go` of the provider the resource identifier is used to _create_ the object in the
cache.

```golang
d := &apps.Deployment{}
nn := app.GetDeploymentNamespacedName(&deployment)

if err := dp.Cache.Create(CoreDeployment, nn, d); err != nil {
    return err
}
```

Notice a new `Deployment` struct is created, along with a namespaced name, and these, together
with the resource identifier, are passed to the `Create()` function. This will create a map in the
resource cache map for this provider resource if it does not already exist, and furnish it with a
key value pair of the namespaced name, and a copy of the deployment retrieved from k8s. It does not
simply create a blank entry, it first tries to obtain a copy from k8s.

The object is then modified, before the following call being made:

```golang
if err := dp.Cache.Update(CoreDeployment, d); err != nil {
    return err
}
```

This call sends the object back to the cache where it is copied.

When another provider wishes to apply updates to this resource, it first needs to retrieve it from the cache. A very similar example may be seen in the
`serviceaccount` provider:

```golang
dList := &apps.DeploymentList{}
if err := sa.Cache.List(deployment.CoreDeployment, dList); err != nil {
    return err
}
for _, d := range dList.Items {
    d.Spec.Template.Spec.ServiceAccountName = app.GetClowdSAName()
    if err := sa.Cache.Update(deployment.CoreDeployment, &d); err != nil {
        return err
    }
}
```

As the resource was created above as a `Multi` resource, the retrieval from the cache must either
use the `List()` function, or the `Get()` function and supply a `NamespacedName`. A *Multi*
resource is one which is expected to hold multiple resources of the same type, but obviously with
different names. If these resources are required to be updated, then an `Update()` call is
necessary on each one as can be seen above.

### Handlers and Watching
This file contains the entrypoints into the watch functions that are used in Clowder. Watches are
used to get Clowder to reconcile when another resource changes. Clowder creates a number of
resources, and example of which is a `Deployment`. When a Deployment that Clowder creates changes,
Clowder needs to know about it so that it can reconcile again and replace the changes if necessary.
Below is an example of multiple watches being set up in the controller.

```golang
watchers := []Watcher{
    {obj: &apps.Deployment{}, filter: deploymentFilter},
    {obj: &core.Service{}, filter: generationOnlyFilter},
    {obj: &core.ConfigMap{}, filter: generationOnlyFilter},
    {obj: &core.Secret{}, filter: alwaysFilter},
}

for _, watcher := range watchers {
    err := r.setupWatch(ctrlr, mgr, watcher.obj, watcher.filter)
    if err != nil {
        return err
    }
}
```

This example sets up four watcher objects with various different filters. There are multiple levels
of filtering that happens to ensure Clowder only reconciles when necessary is to employ the use of
filters. An example of over reconciling would be when a deployment with multiple pods is started, or
redeploys. Every time a pod becomes available, it will change the Deployment resources. This would
ordinarily trigger a reconciliation of ClowdApp, or ClowdEnvironment that owns it. With the correct
filtering in place that doesn't happen.

The `generationOnlyFilter` looks like this:

```golang
func deploymentFilter(logr logr.Logger, ctrlName string) HandlerFuncs {
	return genFilterFunc(deploymentUpdateFunc, logr, ctrlName)
}
```

This creates a new `HandlerFuncs` object that has been configured with the `deploymentUpdateFunc`
object.

```golang
func deploymentUpdateFunc(e event.UpdateEvent) bool {
	objOld := e.ObjectOld.(*apps.Deployment)
	objNew := e.ObjectNew.(*apps.Deployment)
	if objNew.GetGeneration() != objOld.GetGeneration() {
		return true
	}
	if (objOld.Status.AvailableReplicas != objNew.Status.AvailableReplicas) && (objNew.Status.AvailableReplicas == objNew.Status.ReadyReplicas) {
		return true
	}
	if (objOld.Status.AvailableReplicas == objOld.Status.ReadyReplicas) && (objNew.Status.AvailableReplicas != objNew.Status.ReadyReplicas) {
		return true
	}
	return false
}
```

The `genFilterFunc` will take this `deploymentUpdateFunc` and apply it to the `Update` function of
the `HandlerFuncs` object. In this example there are several checks made against the spec and in
certain circumstances, a `true` will be returned. The `true` is an instruction to Clowder to
reconcile objects that are owned by one of Clowder's resources.

There are 4 events that can be triggered when resources change:
* Create
* Update
* Delete
* Generic

The `genFilterFunc` returns an object that contains one of each of these functions.

The functions are then tied to the watcher for a particular type and bound to the
`enqueueRequestForObjectCustom` object in the handlers file. This handler is used for every request
that comes into Clowder. When an Event arrives, the following code will be executed in the example
of a `Create` event.

```golang
func (e *enqueueRequestForObjectCustom) Create(ctx context.Context, evt event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	shouldUpdate, err := e.updateHashCacheForConfigMapAndSecret(evt.Object)
	if err != nil {
		e.logMessage(evt.Object, err.Error(), "", getNamespacedName(evt.Object))
	}

	if shouldUpdate {
		_ = e.doUpdateToHash(evt.Object, q)
		e.reconcileAllAppsUsingObject(ctx, evt.Object, q)
	}

	if own, toKind := e.getOwner(evt.Object); own != nil {
		if doRequest, msg := e.HandlerFuncs.CreateFunc(evt); doRequest {
			e.logMessage(evt.Object, msg, toKind, own)
			q.Add(reconcile.Request{NamespacedName: *own})
		}
	}
}
```

This runs through some special routines that create a hashCache for for serets/configmaps, but
ultimately ends up checking the ownership of the resource to ensure it's owned by the controller
type `ClowdApp` for example. If it is owned by a ClowdApp, the `Create` function that was associated
with the `HandlerFunc` object. If `doRequest` comes back as `true` then the owning Clowder resource
is triggered for reconciliation.


## Commits
We are currently testing [Conventional Commits](https://www.conventionalcommits.org) as a mandatory
step in the pipeline. This requires that each commit to the repo be formatted in the following way:

```
<purpose>(<scope>): <original commit message>

Original multi line commit message
* with items
* in a list
```

An example of this would be:

```
ci(lint): Adds conventional commits

* Adds conventional commits pipeline
* Updates documentation for the contribution to describe the addition of conventional commits
```

It is fine to squash things down into smaller commits - you don't have to have separate commits for everything you add, however, if you do this you **MUST** use the `!` flag appropriately, and it is highly suggested that you use the most significant `purpose` to help describe your change. For example, a change to a provider that adds a new feature and adds a test and documentation could be done as three commits with `feat:`, `test:`, `docs:`, but we will accept a single commit that uses `feat:` because the most significant work in this case is the introduction of a new feature.

### Purposes

Please refer to the [Conventional Commits](https://www.conventionalcommits.org) website for more information. As a quick reference we strongly suggest using the following values for `purpose`:

* ``fix``: For a bug fix
* ``feat``: For a new feature
* ``build``: For anything related to the building of containers
* ``chore``: For any maintenance task
* ``ci``: For anything related to the CI testing pipeline
* ``docs``: For anything related to documentation
* ``style``: For a stylisitic change, usually to adhere to a guideline
* ``refactor``: For any improvements to code structure
* ``perf``: For performance enhancements to code flow
* ``test``: For any changes to tests

Using a `!` after the `purpose/scope` denotes a breaking change in the context of Clowder, this should be used whenever the API for either the Clowd* CRD resources, as well as any change to the `cdappconfig.json` spec. An example of a breaking change is shown below:

```
chore(crd)!: Removes old web field value

* Removes the old web field
```

### Scopes
`scopes` are entirely optional, but using them to indicate changes to providers within clowder would be greatly appreciated. An example is shown below:

```
feat(database)!: Adds new mode for database provider

* Adds MSSQL mode for Clowder
```

## Pull Request Flow

Changes to the Clowder codebase can be broken down into three distinct categories. Each of these
is treated in a slightly different way with regards to signoff and review. The goal of this is to
reduce the size of pull requests to allow code to be merged faster and make production pushes less
dramatic.

* **Typo/Docs** No detailed explanation/justification needed

* **Functional Change** any significant modification to code that gets compiled (i.e. anything over
typo/code style changes) requires a good commit message, detailing functions that have been altered,
behaviour that has changed, etc, a set of functional tests added to the e2e suite, with unit tests
optional, and should be reviewed by at least one Clowder core developer.

* **Architectural Change** anything more advanced than a functional change, which typically
includes, any changes to API specs or changes to external behaviour that is observable by a
customer, should have architect sign off, must be run locally to validate tests and behaviour, must
include any deprecations, should have a design doc, and must be reviewed by two clowder core
developers.

All PRs should be squashed to as few commits as makes sense to a) keep the version history clean
and b) assist with any reverts/cherrypicks that need to happen.

## Testing

Clowder testing utilises two main testing techniques:

* **Unit tests** - small fast tests of individual functions
* **Kuttl/E2E tests** - E2E tests run in a real cluster

The development of tests for these two categories, the sections below detail
some of the development flows for writing tests.

### Types of tests

#### Unit tests

The `controllers/cloud.redhat.com/suite_test.go` is the test file for most of
the top level functions in Clowder. Some providers also have their own test
files to assert specific functionality. This suite does have an etcd process
initiated as part of the test run, but does not have any operators running as
you would expect on a cluster. For example, if a Deployment resource is created
and applied, a Pod resource will NOT be created as it otherwise would be. If
specific functionality is expected to be tested like this, the Kuttl/E2E tests
should be used.

#### Kuttl/E2E tests

The E2E tests make use of the Kuttl suite to test the application and
subsequent result of applying certain resources in a cluster which is running
the Clowder operator. Kuttl applies certain resources, and then asserts that
the resulting resources match those specified. It is suggested to look at the
many examples in the `bundle/tests/scoredcard/kuttl` directory. They are
generally broken down into the following structure. 

```
kuttl/
└── test-name/
    ├── 00-install.yaml
    ├── 01-pods.yaml
    ├── 01-assert.yaml
    ├── 02-json-asserts.yaml
    └── 03-delete.yaml
```

The numerals infront of the test steps define the order Kuttle will invoke
them. The only specially named files are the `*-assert` files, which are
always run last. Sometimes the ordering is forced, e.g. you will usually see
the `delete` files in a separate step at the end to clean up as best it can.

##### `00-install.yaml`

Kuttl usually creates a random namespace for a particular test, but in the
Clowder E2E test suite, the name is required for certain assertions and Kuttl
lacks the means for the E2E suite to reliably retrieve it. The
`00-install.yaml` file usually contains a namespace definition that houses
the test input and output resources.

##### ``01-pods.yaml``

Called `pods` because it will usually contain the definitions that will lead
to pods being created.

##### ``01-assert.yaml``

The resources in this file ill be compared to the ones in the cluster. Kuttl
will wait for a period of time until the resources in the cluster match the
resources in the file. If they do not match when the timeout occurs, the test
is said to have failed.

##### ``02-json-asserts.yaml``

This is a hack as when Kuttl was first introduced it could not run commands as
tests, only as steps in preparing environments. As the inability to complete a
command would halt the test with a failure, the `json-asserts` files are
often used to assert that certain pieces of the JSON (cdappconfig.json) secret
are correct. As these are base64 encoded and contain a blob of data, Kuttl has
no way of matching the resource, so we use the `jq` command to assert
instead.

##### ``03-delete.yaml``

Deletions of the namespace and other resources allow the minikube environment
to be kept as clean as possible during the test run. Leaving pods only
increases resource usage unnecessarily.

### Running tests

To invoke either the unit tests, or the Kuttl tests, the kubebuilder assets are
required to either be on path, or an environment variable needs to be set to
point to them. The example below shows how to run the unit tests by setting the
environment variable.

```shell
KUBEBUILDER_ASSETS=~/kubebuilder_2.3.1_linux_amd64/bin/ make test
```

Running the Kuttl tests requires a cluster to be present. It is possible to run
the Kuttl tests with a simple mocked backplane, but with the complex
integration between multiple operators, the Kuttl tests in Clowder are run
against minikube. With a minikube instance installed and configure as the
default for `kubectl`, the following command will run **all** the e2e tests.

```shell
KUBEBUILDER_ASSETS=~/kubebuilder_2.3.1_linux_amd64/bin/ \
    kubectl kuttl test \
    --config tests/kuttl/kuttl-test.yaml \
    --manifest-dir config/crd/bases/ \
    --manifest-dir config/crd/static/ \
    tests/kuttl/
```

Single tests can be targetted using the `--test` command line flag and using
the name of the directory of the test to be run.
