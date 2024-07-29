# ClowdApp Workflow

Deploying a ClowdApp with new changes should be easy, repeatable, and timely. In
order to make the Clowder experience smoother, there are a few tools to know and
a suggested workflow to follow.

In our general experience, you'll need to make edits to a Clowdapp and your
source code as you work through your Clowder migration. The workflow will
generally follow this pattern: 

1. Make a change in the source code or Clowdapp
2. Deploy the new changes locally
3. Observe the change 
4. Repeat

## 1. Making local changes
Step 1 is pretty case by case basis. Maybe your config is reading the wrong
variable. Perhaps your ClowdApp is missing a Kafka topic. Whatever your changes
may be, update the code and go on to step 2. 

## 2. Deploy locally with Bonfire

Bonfire is a cli tool used to deploy apps with Clowder. Bonfire comes with
a local config option that we'll use to drop our ClowdApp into our minikube
cluster. Read about getting started with bonfire on ephemeral environments [here](https://clouddot.pages.redhat.com/docs/dev/getting-started/ephemeral/index.html)

We'll use our examples from [Getting Started](https://github.com/RedHatInsights/clowder/blob/master/docs/usage/getting-started.rst) again. First, let's make a custom config for our ClowdApp so that bonfire can deploy it without
us needing to push any configuration into app-interface.

Type `bonfire config edit` and add the following to the 'apps' section:

```yaml
apps:
- name: jumpstart
  host: local
  repo: /path/to/your/git/repo
  path: clowdapp.yml
  parameters:
    IMAGE: quay.io/psav/clowder-hello
```

This config instructs bonfire to fetch the template for your application from a git repo located on local disk.

Let's refer back to our example app from earlier. Now, instead of using a base ClowdApp, we will use a Template. 

NOTE: if your previous ClowdApp is still running in the `jumpstart` namespace, use ``oc delete app jumpstart`` to remove it. 

Save ``clowdapp.yml`` in your git repo's folder:

```yaml
apiVersion: v1
kind: Template
metadata:
  name: jumpstart
objects:
- apiVersion: cloud.redhat.com/v1alpha1
  kind: ClowdApp
  metadata:
    name: hello
    namespace: jumpstart
  spec:

    # The bulk of your App. This is where your running apps will live
    deployments:
    - name: app
      # Creates services based on the ports set in ClowdEnv
      webServices:
        public: true
        private: false
        metrics: true
      # Give details about your running pod
      podSpec:
        image: ${IMAGE}:${IMAGE_TAG}

    # The name of the ClowdEnvironment providing the services
    envName: ${ENV_NAME}
    
    # Request kafka topics for your application here
    kafkaTopics:
      - replicas: 3
        partitions: 64
        topicName: topicOne

    # Creates a database if local mode, or uses RDS in production
    database:
      # Must specify both a name and a major postgres version
      name: jumpstart-db
      version: 12

parameters:
  - name: IMAGE
    value: quay.io/psav/clowder-hello
  - name: IMAGE_TAG
    value: latest
  - name: ENV_NAME
    required: true
```


Now run:

```shell
bonfire deploy jumpstart -n jumpstart
```

## 3. Observe the changes
Run ``kubectl get app -n jumpstart`` to verify the jumpstart app has been deployed.

If the deployment fails to reach a 'ready' state or pods fail to come up, use ``kubectl get events -n jumpstart`` and look for any errors related to your ClowdApp. You can also use ``kubectl logs <pod name> -n jumpstart --previous=true`` if your pods are crash looping.

## 4. Repeat
Repeat until you're happy with the results. When satisfied, checkout the
[migration guide](../migration/migration.md) to start your app on the jouney to ephemeral and beyond.   


## Next Steps
[Migrating a service from v3 to Clowder](../migration/migration.md)
