# Frequently Asked Questions

This document is intended to give answers to some of the more common question surrounding the use of
Clowder. If a question that you think should be included here has been omitted, please raise a
ticket and the team will do their best to answer it swiftly.

## Operations

### Is there an easy way to see what is happening with all my ClowdApps/ClowdEnvironment?

On the console, listing the apps in a namespace by using ``oc get app`` should give enough
information about the ready state of each app. If you need more information, then the ``status``
section of the resource should provide conditions detailing why a reconciliation has failed. For the
more visual developer, there is a link at the bottom of the ephemeral env which will show
ClowdApp/ClowdEnvironment status on a page and will auto update. This is expected to land in
stage/prod soon.

### Whenever I make changes to a deployment, they just get overwritten, why is this?

This is a core concept in Clowder that we should always be ensuring that the deployment be kept
inline with the application's specification. By making changes to a deployment directly on a
cluster, it's possible that changes will be introduced that will affect its operation. Thus, Clowder
tries to ensure that any such changes are reverted as quickly as possible. If changes are needed to
be made to a deployment the following options are available.

* Make a change to the ClowdApp
* Temporarily disable Clowder's control of the application by setting the [`spec.disabled`](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-clowdapp) flag.

NOTE: Using app-sre's tooling changes to resources managed by app-interface would be changed in much
the same way as Clowder, except at a much slower rate, say once every 15 minutes. Clowder watches
many resources and will respond to changes instantaneously.

### How can I add multiple containers to my deployment's pod?

In short, you can't. One of the core tenets of Clowder is to create homogenous deployment setups, in
this regard a particular pod will only have one single configurable container. That is not to say
that a pod will never have multiple containers. The use of the ephemeral environment and sidecar
features do add other sidecars to the pod, but Clowder restricts an app to one application container
per pod.

### How should I set up liveness and readiness probes?

Clowder tries to use the Kubernetes best practice of listening to the ``/healthz`` endpoint on the
``webPort`` which has been defined (default 8000). If this does not exist because either the app has
decided on a different approach or has not yet created a _healthz_ endpoint, then the liveness and
readiness probes can be overridden using the
[`spec.deployments.podSpec.livenessProbe`](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-podspec)
and
[`spec.deployments.podSpec.readinessProbe`](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-podspec)
stanzas. These follow the same structure as normal Kubernetes liveness/readiness probes.

NOTE: You should always use port names like `web`, `private` or `metrics` in your probes, as the
port values can be changed by Clowder. Using a name insulates the application from these changes.

## Database

### How can I share a database from another ClowdApp?

Clowder allows you to share an app's database credentials to secondary apps. One app will be the
_owner_ of the database and would be responsible for performing migrations etc, other apps can then
consume those credentials. Currently secondary apps consuming those credentials will get full write
access to the database and are responsible for ensuring they do not negatively affect it.

To set this up, use the [`spec.providers.database.sharedDbAppName`](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-databasespec).

NOTE: The `spec.providers.database.name` cannot be set with this feature. App interface mode also
supports this feature. `spec.providers.database.version` is ignored in shared db mode.

### How can I migrate to a new database?

Clowder only supports connection to one database at a time. In *app-interface* mode, this can be
problematic as there are occasions where a database snapshot restore is necessary. When performing
this the app effectively needs to connect to a new database. As a part of this setup, the new
database will receive a new hostname. With the legacy way of choosing database secrets in
*app-interface* mode, this would mean changing the name of the database in the ClowdApp stanza
which will end up causing much disruption. 

In Clowder 0.16.0, a new way of declaring database secrets in *app-interface* mode was added, this
makes use of annotations to decide which database credential to expose to the application. This
takes the form of the following example and should be applied in app-interface

```yaml
terraformResources:
  - name: my_resource
    annotations:
      clowder/database: app-d
```

In this example, the ClowdApp would have the database name set to ``app-d``. To migrate to a new
database, bring up the second secret with the annotation applied to it, and remove it from the
first.


## Web Services

### How do I set up an internal port for inter-app communication?

To allow two apps to talk together internally without exposing a port to the public, use the
[`spec.deployments.webServices.private`](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-privatewebservice)
configuration stanza.

This will enable the use of a private port that will not be exposed via public mechanisms.

### Can I disable the metrics port?

No. Clowder will always set up a metrics port and service for you. You can of course choose not to
run any service on that port, but then when Clowder manages service monitors, it will have nothing
to monitor and alerts will probably fail. It is seen to be a best practice for an application to
always have a metrics port and serve some kind of prometheus scrapable metrics, however limited.

## Storage

### Can I enable PVCs for my pods?

You _can_, but you should really consider if this is the best option. Clowder allows for the use of
many infrastructure storage options, kafka, rds, s3, elasticache, using a PVC should be considered a
last resort. Clowder has no way to specifically request a PVC either, this must be done outside the
ClowdApp and referenced using the volumes portion of the deployment spec.

## Misc

### How should I set up dependencies?

Though there is talk about changing the definition at a later date, dependencies currently work with
these assumptions.

* A **Dependency** listed in the
[`spec.dependencies`](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-podspec)
section of a ClowdApp lists those dependencies which are required for the application to run. That
is to say if the dependency is not present, the application will fail to start properly, or perform
at all.
* An **Optional Dependency** listed in the
[`spec.dependencies`](https://redhatinsights.github.io/clowder/clowder/dev/api_reference.html#k8s-api-github-com-redhatinsights-clowder-apis-cloud-redhat-com-v1alpha1-podspec)
section is one which the application can tolerate a failure of gracefully. The application will also
be restarted should one of its _optional dependencies_ appear or disappear from the cluster. This is
because the configuration for an _optional dependency_ is added to or removed from the
``cdappconfig,json`` in line with the presence of the application on the cluster.

## Can I have two different applications using two different provider modes?

No. Currently Clowder is not able to differentiate different modes for different apps. This is
largely because it would be for a very special use case and would be difficult to define. A key
concept of the ClowdApp is that it embodies all the information required to run the app, regardless
of environment. Having to define environmental modes in the ClowdApp would break this model. It is
possible they could be added to a section on the ClowdEnvironment, but this is not a feature on the
roadmap currently.
