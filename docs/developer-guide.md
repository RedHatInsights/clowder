# Clowder Environment Setup

## Dev Environment Setup

* Install the latest [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* Install [krew](https://krew.sigs.k8s.io/docs/user-guide/setup/install/)
* Install kuttl:
  ``kubectl krew install kuttl``
* Install [kubebuilder](https://book.kubebuilder.io/quick-start.html#installation)

  * Normally kubebuilder is installed to ``/usr/local/kubebuilder/bin``. You should see the following
  executables in that directory:
    
    ``etcd  kube-apiserver  kubebuilder  kubectl``

    You may want to append this directory to your ``PATH`` in your ``.bashrc``, ``.zshrc``, or similar.

    NOTE: If you choose to place the kubebuilder executables in a different path, make sure to
    use the ``KUBEBUILDER_ASSETS`` env var when running tests (mentioned in ``Unit Tests`` section below)

* Install https://kubectl.docs.kubernetes.io/installation/kustomize/binaries/[kustomize]
** The install script places a ``kustomize`` binary in whatever directory you ran the above script in. Move this binary to a folder that is on your ``PATH`` or make sure the directory is appended to your ``PATH``

* Install https://minikube.sigs.k8s.io/docs/start/[minikube]. The latest release we have tested with is https://github.com/kubernetes/minikube/releases/tag/v1.20.0[v1.20.0].

NOTE: If you want/need to use OpenShift, you can install [Code Ready Containers](https://github.com/RedHatInsights/clowder/blob/master/docs/crc-guide.md), just be aware that it consumes a much larger amount of resources and our test helper scripts are designed to work with minikube.

We haven't had much success using the docker/podman drivers, and would recommend the [kvm2 driver](https://minikube.sigs.k8s.io/docs/drivers/kvm2/) or [virtualbox driver](https://minikube.sigs.k8s.io/docs/drivers/virtualbox/)

### **KVM2-specific notes**

* If you don't have virtualization enabled, follow the guide
  on https://docs.fedoraproject.org/en-US/quick-docs/getting-started-with-virtualization/[the minikube docs]

* Note that ``virt-host-validate`` may throw errors related to cgroups on Fedora 33 -- which you can https://gitlab.com/libvirt/libvirt/-/issues/94[ignore]

* If you don't want to enter a root password when minikube needs to modify its VM, add your user to the ``libvirt`` group:

  ```shell
  sudo usermod -a -G libvirt $(whoami)
  newgrp libvirt
  ```

* You can set minikube to default to kvm2 with: ``minikube config set driver kvm2``

* Move to the ``Testing`` section below to see if you can successfully run unit tests and E2E tests.

## Running

* ``make install`` will deploy the CRDs to the cluster configured in your kubeconfig.
* ``make run`` will build the binary and locally run the binary, connecting the
  manager to the Openshift cluster configured in your kubeconfig.
* ``make deploy`` will try to run the manager in the cluster configured in your
  kubeconfig.  You likely need to push the image to a docker registry that your Kubernetes
  cluster can access.  See `E2E Testing` below.
* ``make genconfig`` (optionally) needs to be run if the specification for the config
  has been altered.

## Using a Debugger
Developing Clowder is easier if you run the code in a debugger. We'll cover two ways to do this: with Delve and with VS Code. Both provide the same features (VS Code actually uses Delve under the hood.) The difference is VS Code provides you with a GUI and integration with the IDE whereas Delve is a command line tool and required using the Delve CLI and command system.

### Pre-requisites
Wether you choose to use Delve or VS Code, you'll need to set up a few things first. You'll need to start minikube and build and install the CRDs. You'll also need a Clowder config file handy.

* Ensure you have clowder code cloned and you've installed the CRDs into minikube 

  ```shell
  minikube start --cpus 4 --disk-size 36GB --memory 16000MB --driver=kvm2 --addons registry --addons ingress --addons metrics-server ; make build ; make install ; ./build/kube_setup.sh
  - Ensure you have a clowder config file saved. Here's an example config file:
  [source,json]
  {
      "debugOptions": {
          "trigger": {
              "diff": false
          },
          "cache": {
              "create": false,
              "update": false,
              "apply": false
          },
          "pprof": {
              "enable": true,
              "cpuFile": "testcpu"
          }
      },
      "features": {
          "createServiceMonitor": true,
          "watchStrimziResources": true,
          "enableKedaResources": true,
          "reconciliationMetrics": true,
          "enableExternalStrimzi": true,
          "disableWebhooks": true
      }
  }
  ```

### Delve
Delve is a Go debugger. It allows you to run your app, set breakpoints, etc from the command line. Further reading will be required to learn to use Delve effectively if you haven't used Delve before.

* Run the through the Pre-requisites section above and make sure you have minikube running and the CRDs installed.
* Ensure you have [Delve](https://github.com/go-delve/delve), the Go debugger installed
* Now you can start the debugger, while setting the CLOWDER_CONFIG_PATH environment variable to the path of your config file. After launching the debugger you can set any breakpoints you want and then continue execution.

  ```shell
  $ CLOWDER_CONFIG_PATH=/path/to/config/sample_config.json  /go/path/dlv debug ./main.go
  Type 'help' for list of commands.
  (dlv) continue
  Loading config from: /home/adamdrew/clowderConfig/sample_config.json
  {"level":"info","ts":1664213244.8772495,"logger":"setup","msg":"Loaded config","config":{"images":{"mbop":"","caddy":"","Keycloak":"","mocktitlements":""},"debugOptions":{"Logging":{"debugLogging":false},"trigger":{"diff":false},"cache":
  ```

### VS Code
VS Code an open source IDE that is popular with many of the Clowder developers. Setting up a debugger with VS Code is easy and provides a GUI for setting breakpoints, stepping through code, etc.

* Run the through the Pre-requisites section above and make sure you have minikube running and the CRDs installed.
* Install https://code.visualstudio.com/[VS Code]
* Install the https://marketplace.visualstudio.com/items?itemName=golang.Go[Go extension] for VS Code
* Open the Clowder code in VS Code
* Create a launch.json file in the .vscode directory. Here's an example launch.json file:

  ```json
  {
      "version": "0.2.0",
      "configurations": [
          {
              "name": "Clowder Launch",
              "type": "go",
              "request": "launch",
              "mode": "auto",
              "program": "${workspaceFolder}/",
              "env": {
                  "CLOWDER_CONFIG_PATH": "/home/adamdrew/clowderConfig/sample_config.json"
                }
          },
      ]
  }
  ```

* You can now debug Clowder in VS Code

## Testing

### Unit Testing

The tests rely on the test environment set up by controller-runtime.  This enables the operator to 
get initialized against a control plane just like it would against a real OpenShift cluster.

To run the tests:

``make test``

If kubebuilder is installed somewhere other than ``/usr/local/kubebuilder/bin``, then:
``KUBEBUILDER_ASSETS=/path/to/kubebuilder/bin make test``

If you're just getting started with writing tests in Go, or getting started with Go in general, take
a look at https://quii.gitbook.io/learn-go-with-tests/

### Debugging Unit Tests in VS Code
We include a special make target that will launch a special debug instance of VS Code that can run and debug the unit tests. This is useful if you want to set breakpoints in the unit tests and step through them.

* Make sure you have all of the pre-requisites for running clowder installed and operational.
* Make sure you have VS Code installed including the Go extension.
* Close any instance of VS Code that has Clowder open in it.
* Run the following command:

  ```shell
  make vscode-debug
  ```

  - VS Code will launch
  - Open any file that contains unit tests
  - You will see the green play button next to each test as well as the `run test` and `debug test` buttons above each test
  - You can use the run or debug test buttons to run and debug the tests from within VS Code

  NOTE: Some features of VS Code may not work correctly when launched this way. We reccomend only launching code this way when you want to write and debug unit tests.

### E2E Testing

There are two e2e testing scripts which:

* build your code changes into a docker image (both ``podman`` or ``docker`` supported)
* push the image into a registry
* deploy the operator onto a kubernetes cluster
* run `kuttl`` tests

The scripts are:

* ``e2e-test.sh`` -- pushes images to quay.io, used mainly for this repo's CI/CD jobs or in cases where you have
  access to a remote cluster on which to test the operator.
* ``e2e-test-local.sh`` -- pushes images to a local docker registry, meant for local testing with minikube

You will usually want to run:

```shell
minikube start --addons=registry --addons=ingress  --addons=metrics-server --disable-optimizations
/e2e-test-local.sh
```

### Podman Notes
If using podman to build the operator's docker image, ensure sub ID's for rootless mode are configured:
Test with:

```shell
podman system migrate
podman unshare cat /proc/self/uid_map
```

If those commands throw an error, you need to add entries to ``/etc/subuid`` and ``/etc/subgid`` for your user.
The subuid range must not contain your user ID and the subgid range must not contain your group ID. Example:

```shell
❯ id -u
112898
❯ id -g
112898

# Use 200000-265535 since 112898 is not found in this range
❯ sudo usermod --add-subuids 200000-265535 --add-subgids 200000-265535 $(whoami)

# Run migrate again:
❯ podman system migrate
❯ podman unshare cat /proc/self/uid_map
```

## Migrating an App to Clowder

[Insights App Migration Guide](migration/migration.md)

## Doc generation

### Prerequisites

The API docs are generated by using the [crd-ref-docs](https://github.com/elastic/crd-ref-docs) tool
by Elastic. You will need to install ``asciidoctor``:

On Fedora use:

``sudo dnf install -y asciidoctor``

For others, see: https://docs.asciidoctor.org/asciidoctor/latest/install/


### Generating docs

Generating the docs source using:

``make api-docs``

Then be sure to add doc changes before committing, e.g.:

``git add docs/api_reference.md``

## Clowder configuration

Clowder can read a configuration file in order to turn on certain debug options, toggle feature
flags and perform profiling. By default clowder will read from the file
``/config/clowder_config.json`` to configure itself. When deployed as a pod, it an optional volume
is configured to look for a ``ConfigMap`` in the same namespace, called ``clowder-config`` which
looks similar to the following.

```yaml
apiVersion: v1
data:
  clowder_config.json: |-
    {
        "debugOptions": {
            "trigger": {
                "diff": false
            },
            "cache": {
                "create": false,
                "update": false,
                "apply": false
            },
            "pprof": {
                "enable": true,
                "cpuFile": "testcpu"
            }
        },
        "features": {
            "createServiceMonitor": false,
            "watchStrimziResources": true
        } 
    }
kind: ConfigMap
metadata:
  name: clowder-config
```

To run clowder with the ``make run`` (or to debug it VSCode), and apply configuration, it is
required to either create the ``/config/clowder_config.json`` file in the filesystem of the machine
running the Clowder process, or to use the environment variable ``CLOWDER_CONFIG_PATH`` to point to
an alternative file.

At startup, Clowder will print the configuration that was read in the logs

```text
[2021-06-16 11:10:44] INFO   Loaded config config:{'debugOptions': {'trigger': {'diff': True}, 'cache': {'create': True, 'update': True, 'apply': True}, 'pprof': {'enable': True, 'cpuFile': 'testcpu'}}, 'features': {'createServiceMonitor': False, 'disableWebhooks': True, 'watchStrimziResources': False, 'useComplexStrimziTopicNames': False}}
```

### Debug flags

Clowder has several debug flags which can aid in troubleshooting difficult situations. These are 
defined in the below.

* ``debugOptions.trigger.diff`` - When a resource is responsible for triggering a reconciliation of
  either a ``ClowdApp`` or a ``ClowdEnvironment`` this option will print out a diff of the old and 
  new resource, allowing an inspection of what actually triggered the reconciliation.

  ```text
  [2021-06-16 11:24:49] INFO APP  Reconciliation trigger name:puptoo-processor namespace:test-basic-app resType:Deployment type:update
  [2021-06-16 11:24:49] INFO APP  Trigger diff diff:--- old
  +++ new
  @@ -3,8 +3,8 @@
      "name": "puptoo-processor",
      "namespace": "test-basic-app",
      "uid": "de492af3-be26-4a2c-b959-54b674c9e34f",
  -    "resourceVersion": "43162",
  -    "generation": 1,
  +    "resourceVersion": "44111",
  +    "generation": 2,
      "creationTimestamp": "2021-06-16T10:19:20Z",
      "labels": {
        "app": "puptoo",
  @@ -69,7 +69,7 @@
          "manager": "manager",
          "operation": "Update",
          "apiVersion": "apps/v1",
  -        "time": "2021-06-16T10:19:20Z",
  +        "time": "2021-06-16T10:24:49Z",
          "fieldsType": "FieldsV1",
          "fieldsV1": {
            "f:metadata": {
  ```

* ``debugOptions.cache.create`` - When an item is *created* in Clowder's resource cache, this option
  will enable printing of the resource that came from k8s cache. If the resource exists in k8s, this 
  will be the starting resource that Clowder will update.

  ```
  [2021-06-16 11:20:23] INFO  [test-basic-app:puptoo] CREATE resource  app:test-basic-app:puptoo diff:{
  "kind": "Deployment",
  "apiVersion": "apps/v1",
  "metadata": {
    "name": "puptoo-processor",
    "namespace": "test-basic-app",
    "uid": "de492af3-be26-4a2c-b959-54b674c9e34f",
    "resourceVersion": "43162",
    "generation": 1,
    "creationTimestamp": "2021-06-16T10:19:20Z",
    "labels": {
      "app": "puptoo",
      "pod": "puptoo-processor"
  ...
  ...
  ```

* ``debugOptions.cache.update`` - When enabled, and an item is *updated* in Clowder's resource 
  cache, this option will print the new version of the item in the cache.

  ```text
  [2021-06-16 11:20:23] INFO  [test-basic-app:puptoo] UPDATE resource  app:test-basic-app:puptoo diff:{
  "kind": "ServiceAccount",
  "apiVersion": "v1",
  "metadata": {
    "name": "iqe-test-basic-app",
    "namespace": "test-basic-app",
    "uid": "3d89ab16-dcb2-4dbb-b0e6-685009878175",
    "resourceVersion": "43135",
    "creationTimestamp": "2021-06-16T10:19:20Z",
    "labels": {
      "app": "test-basic-app"
    },
    "ownerReferences": [
      {
        "apiVersion": "cloud.redhat.com/v1alpha1",
        "kind": "ClowdEnvironment",
        "name": "test-basic-app",
        "uid": "25e121df-5b12-4c34-b8f3-a49b0f20afcf",
        "controller": true
      }
    ],
  ```


* ``debugOptions.cache.apply`` - This option is responsible for printing out a diff showing what the
  resource was when it was first read into Clowder's cache via the ``create``, and what is being 
  applied via the k8s client.

  ```
  [2021-06-16 11:20:23] INFO  [test-basic-app:puptoo] Update diff app:test-basic-app:puptoo diff:--- old
  +++ new
  @@ -84,14 +84,14 @@
          "protocol": "TCP",
          "appProtocol": "http",
          "port": 8000,
  -        "targetPort": 0
  +        "targetPort": 8000
        },
        {
          "name": "metrics",
          "protocol": "TCP",
          "appProtocol": "http",
          "port": 9000,
  -        "targetPort": 0
  +        "targetPort": 9000
        }
      ],
  ```


* ``debugOptions.pprof.enable`` - To aid in profiling, this option enables the cpu profilier.
* ``debugOptions.pprof.cpuFile`` - This option sets where the cpu profiling saves the collected 
  pprof data.

### FeatureFlags
Clowder currently support several feature flags which are intended to enable or disable certain 
behaviour. They are detailed as follows:

| Flag Name | Description | Permanent
|--|--|--
| ``features.createServiceMonitor`` | Enables the creation of prometheus ``ServiceMonitor`` resources. | No
| ``features.disableWebhooks`` | While testing locally and for the ``suite_test``, the webhooks need to be disabled. this option facilitates that. | Yes
| ``features.watchStrimziResources`` | When enabled, Clowder will assume ownership of the ``Kafka`` and ``KafkaConnect`` resources it creates. It will then respond to changes to these resources. | No
| ``features.useComplexStrimziTopicNames`` | This flag switches Clowder to use non-colliding names for strimzi resources. This is important if using a singular strimzi server for multiple ``ClowdEnvironment`` resources. | Yes
| ``features.enableAuthSidecarHook`` | Turns the sidecar functionality on or off globally. | Yes
| ``features.enablekedaResources`` | Turns on the addition of Keda resources into the protectedGVK list. | No
| ``features.perProviderMetrics`` | Turns on metrics that calculate reconciliation time per provider. | Yes
| ``features.reconciliationMetrics`` | Enables extra detailed metrics on reconciliations per application. | Yes
| ``features.enableDependencyMetrics`` | Turns on metrics that report availability of a ClowdApps dependencies. | Yes
| ``disableCloudWatchLogging`` | Disables logging to CloudWatch. | Yes
| ``enableExternalStrimzi`` | Enables talking to Strimzi via a local nodeport (only useful on minikube) | Yes
| ``disableRandomRoutes`` | Gives the ability to disable the extra portion of randomness added to routes. | Yes
