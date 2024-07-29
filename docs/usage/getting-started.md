# Getting Started

## Install Clowder on your local box

As covered in the [main README section](../index.md#getting-clowder), the installation process for
Clowder is a breeze.

Using the above method will install Clowder locally through your Minikube
instance. It will also create two new custom resource types that are easy to
query: ``env`` for ``ClowdEnvs`` and ``app`` for ``ClowdApps``.

If you would like to install Clowder manually and set up a contributing
developer environment, [follow the developer guide](../developer-guide.md).

## Create your first ``ClowdEnvironment``

Let's make a namespace to hold all the resources we'll be creating :

```shell
kubectl create ns jumpstart
```

That's the example namespace we'll be using for the rest of the guide.

Now we can drop in our first resource, a ClowdEnvironment. A ClowdEnvironment,
or ClowdEnv, is a custom resource that defines the environment our ClowdApp will
utilize. The ClowdEnv defines what types of services our app may require and
what source is providing those services. For our purposes, these services are configured to operate with ephemeral environments.

The easiest way to install a ClowdEnvironment that is fit for working with ephemeral environments is by using Bonfire.

Bonfire is a cli tool used to deploy apps into ephemeral environments. Read about getting started with bonfire on ephemeral environments [here](https://consoledot.pages.redhat.com/docs/dev/creating-a-new-app/using-ee/index.html)

If you're curious to see what is going to be deployed (without actually applying the config), run:

```shell
bonfire process-env -n jumpstart
```

You can apply the environment's config and wait for it to become "ready" using:

```shell
bonfire deploy-env -n jumpstart
```

This will cause bonfire to apply the [default ephemeral template](https://github.com/RedHatInsights/bonfire/blob/master/bonfire/resources/ephemeral-cluster-clowdenvironment.yaml) and set the ``targetNamespace`` to ``jumpstart```

NOTE: You will only create a ClowdEnvironment in your local minikube. Stage
and Production will have one ClowdEnv, respectively, shared by all apps in
that environment. The ephemeral cluster has a ClowdEnvironment created for you ahead of time when you reserve a namespace.

Let's see what the ClowdEnv does.

```shell
kubectl get env env-jumpstart -o yaml
```

As you can see in the output, we have ``providers``_ for the different services. Some of these providers have caused certain deployments to appear in the environment's ``targetNamespace`` such as kafka, minio, featureflags service, etc.
These will be used by ClowdApps associated with this environment.

### Accessing services running inside your namespace

Pods running within your kubernetes cluster can access the services set up by the ClowdEnvironment. However, if you are wanting to access a ClowdEnv-provided service (Kafka, Minio, etc) directly from your host we need edit to your ``/etc/hosts`` mappings. Our example uses
Kafka, so we are going to look up the bootstrap service address in our environment:

```shell
kubectl get kafka -n jumpstart -o jsonpath='{.items[0].status.listeners[0].bootstrapServers}' | cut -f1 -d':'
```

This is your kafka cluster's boostrap FQDN. It will look similar to:

```
env-jumpstart-cadf501e-kafka-bootstrap.jumpstart.svc
```

Your ``/etc/hosts`` should now look like ::

```
127.0.0.1   localhost ...  env-jumpstart-cadf501e-kafka-bootstrap.jumpstart.svc.
```

You can then use ``kubectl port-forward svc/env-jumpstart-cadf501e-kafka-bootstrap -n jumpstart 9092:9092`` and access the kafka broker using ``env-jumpstart-cadf501e-kafka-bootstrap.jumpstart.svc:9092``

## Create your first ClowdApp

Now that we have a ClowdEnv up and running, let's use those providers and get
some pods going. We can do that using a ClowdApp. You can think of a ClowdApp
much like a Deployment resource, but more powerful. In your ClowdApp, you define
everything your app needs to run: database names, object storage, environment
variables, container images, and cron jobs; the whole party. We'll start small
and use the example.

The [API docs for ClowdApps](https://consoledot.pages.redhat.com/clowder/dev/api_reference.html) can be found on redhatinsights.github.io.

A [fully annotated ClowdApp](../examples/clowdapp.yml) file can be found in the Clowder examples directory.

Now let's add our ClowdApp:

```shell
kubectl apply -f https://raw.githubusercontent.com/RedHatInsights/clowder/master/docs/examples/clowdapp.yml -n jumpstart
```

Let's verify that ClowdApp was created:

```shell
kubectl get app -n jumpstart
```

Now you should see pods!:

```shell
kubectl get pods -n jumpstart -w
```

This should show you several running pods. Some of them we defined in our
ClowdApp, some we did not. Pods like Kafka are defined in the ClowdEnv and spun
up when the environment was created, while others are related to your ClowdApp. As a note, your app will not come up until the all ClowdEnv's managed deployments are marked as ready.

That's it! You have a running ClowdApp deployed with Clowder. In the next few
documents, we'll cover creating a more powerful dev environment, building a more
complex ClowdApp, and migrating existing services over to Clowder.
