Developing for Clowder
======================

Providers
---------

Provider setup
++++++++++++++

The core of Clowder revolves around a core concept of providers. Clowder splits its functionality
into multiple units called providers. These are called in a defined order and provide both
environmental and application level resources, as well as configuration that ultimately lands in the
``cdappconfig.json``.

The providers live in the ``controllers/cloud.redhat.com/providers`` folder and comprise of a 
``provider.go`` file, some mode files, some implementation files and potentially some tests too.

The ``provider.go`` defines several key pieces. Shown below is the ``deployment`` provider's
``provider.go`` file:

.. code-block:: go

    package deployment

    import (
        p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
        apps "k8s.io/api/apps/v1"
    )

    // ProvName sets the provider name identifier
    var ProvName = "deployment"

    // CoreDeployment is the deployment for the apps deployments.
    var CoreDeployment = p.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})

    // GetEnd returns the correct end provider.
    func GetDeployment(c *p.Provider) (p.ClowderProvider, error) {
        return NewDeploymentProvider(c)
    }

    func init() {
        p.ProvidersRegistration.Register(GetDeployment, 0, ProvName)
    }

The ``ProvName`` is an identifier that defines the name of the provider. Notice that the Golang
pacakge name is the same as this identifier. This is a nice convention and one which should be
maintained when new providers are added. The next declaration is a MultiResourceIdent. These will be
discussed in a little mroe detail below, but in short, this is a declaration of the resources that
this particular provider will create.

After that there is the ``GetDeployment()`` function. Every provider has some kind of ``Get*()``
function, which is responsible for creating deciding which mode to run the provider in. Depending on
the environmental settings, providers can be run in different modes. The ``deployment`` provider is
a core provider and as such as no modal configuration, i.e. there is only one mode. Providers with
no modes will use the ``default.go`` to provide their functionality. The ``Get*()`` function returns
a Provider object. In this case the function is called ``NewDeploymentProvider()`` and returns the
default ``DeploymentProvider`` object. This will be expanded upon shortly.

The ``init()`` call is responsible for registering this provider with the internal provider
registration system. The provider's ``Get*()`` function is passed in, as well as an integer and the
``ProvName``. The integer specifies the order in which the providers should be run. ``0`` is the
first provider and ``99``, by convention, is the last. Two providers can share the same order
number.

Care must be taken when providers depend on each others resources, that they are executed in the
correct order, otherwise the dependant provider will fail when its dependency is missing from the
cache. This will be explained in more depth in the caching section later in this document.

Provider functionality
++++++++++++++++++++++

The ``default.go`` file is shown below:

.. code-block:: go

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

The ``default.go`` file defines a singular mode for a provider. In other providers there may be
several modes and each of these will be housed in its own ``.go`` file, though it will be a part of
the same package. The ``deploymentProvider`` struct defines an struct to which functions are
attached for provider operation. Some of these can be internal, but the most important one is called
``Provide`` and must be exported.

When the providers are _invoked_ in Clowder, they are done so in the two controllers,
``ClowdEnvironment`` and ``ClowdApp``. The ``ClowdEnvironment`` controller only runs the
_environmental_ functionality to provider environmental resources. An example of this would be a
kafka or obejct storage server, as there is only ever one of these per environment. The
``NewDeploymentProvider()`` function, as referenced in the previous ``provider.go`` file, is
responsible for creating and managing these _environment_ level resources. These are run by the
_environment_ controller and will be reconciled whenever the ``ClwodEnvironment`` is triggered.

By contrast, ``ClowdApp`` modifications trigger the _application_ reconciliation, which first runs
the _environment_ function, in this case ``NewDeploymentProvider()`` before then running the
``Provide()`` function. This may seem odd and indeed is a design quirk of Clowder that iwill
hopefully be resolved in a later release. Its reasoning is that the environmental resources often
need to provide information to the application level reconciliation, for instance to furnish the
``cdappconfig`` with the Kafka broker address. Since this information is calculated by the
environment controller, the application controller must first rerun the environment controller's
functions to obtain the information.

Environment and application level functions can access and edit the ``AppConfig`` object which will
ultimately be transformed into the ``cdappconfig.json`` file that ends up in the app container at
runtime.

Caching resources
-----------------

A key tenet of the Clowder provider system is that of sharing resources. Without resource sharing,
providers that need to modify the resources of other providers result not only in multiple calls to
update the same resources, but also can potentially trigger multiple reconciliations as updates to
Clowder owned resources can trigger these.

To reduce this burden, the Clowder system will onyl apply resources at the very end of the
reconciliation. Until that time, resources are stored in the resource cache and providers are able
to retrieve objects from this cache, update them, and then placed the updated versions back in the
cache, so that their changes will be applied at the end of the reconciliation. This is where the
order of provider invocation is important.

The following is a snippet from the ``deployment`` provider's ``provider.go``:

.. code-block:: go

    // CoreDeployment is the deployment for the apps deployments.
    var CoreDeployment = p.NewMultiResourceIdent(ProvName, "core_deployment", &apps.Deployment{})

This was shown previously and is responsible for creating an object that can identify certain
resources. The call takes three arguments: the provider name, a purpose string (which details
briefly what the resource is used for), and a _template_ object.

.. note::

    The template object is never *used* in anyway. It is merely there to determine the type of the
    resource.

In the ``impl.go`` of the provider the resource identifier is used to _create_ the object in the
cache.

.. code-block:: go

    d := &apps.Deployment{}
    nn := types.NamespacedName{
        Name:      GetDeploymentName(app, &deployment),
        Namespace: app.Namespace,
    }

    if err := dp.Cache.Create(CoreDeployment, nn, d); err != nil {
        return err
    }

Notice a new ``Deployment`` struct is created, along with a namespaced name, and these, together
with the resource identifier, are passed to teh ``Create()`` function. This will create a map in the
resource cache map for this provider resource if it does not already exist, and furnish it with a
key value pair of the namespaced name, and a copy of the deployment retrieved from k8s. It does not
simply create a blank entry, it first tries to obtain a copy from k8s.

The object is then modified, before the following call being made:

.. code-block:: go

	if err := dp.Cache.Update(CoreDeployment, d); err != nil {
		return err
	}

This call sends the object back to the cache where it is copied.

When another provider wishes to apply updates to this resource, it first needs to retrieve it from the cache. A very simliar example may be seen in the
``serviceaccount`` provider:

.. code-block:: go

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

As the resource was created above as a ``Multi`` resource, the retrieval from the cache must either
use the ``List()`` function, or the ``Get()`` function and supply a ``NamespacedName``. A *Multi*
resource is one which is expected to hold multiple resources of the same type, but obviously with
different names. If these resources are required to be updated, then an ``Update()`` call is
necessary on each one as can be seen above.
