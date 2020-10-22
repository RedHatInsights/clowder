# Running Clowder on CRC 

## Prerequisites
-----------------
1. Download the [crc binary](https://developers.redhat.com/products/codeready-containers/overview) and follow the instructions to get crc running.
2. Fork or clone the [Clowder repo](https://github.com/RedHatInsights/clowder)
3. Install [the Clowder dependencies](https://github.com/RedHatInsights/clowder#dependencies)
4. Run `make install`


## Running Clowder
-------------------
At this time it is not recommended to deploy Clowder to your crc cluster. Instead, we will run the Operator on your local machine. To do this, we need add the clowder-system services to your `/etc/hosts` localhost (127.0.0.1). For this example, we are using the `ingress-env-minio.clowder-system.svc` service because it matches our environment's name. Follow the Kubernetes service pattern for whatever your entry may need to be; just be sure it matches your specific environment name. 

Your `etc/hosts` should now look like `127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4 ingress-env-minio.clowder-system.svc`. If you are not using the ingress-env, change it to the appropriate service. 

We're going to use Ingress as the example, so the configuration we're doing is specific to that. If you are standing up a different application, substitute your own services, or other variables. 

`make run 2>&1 | grep '^{' | jq -r .`

This will start the operator on your local machine with output redirected to `jq`. The jq output is formatted and easier to read and therefore recommended. However, if don't want that, `make run` will do just fine; albeit less neat. 

## Applying the ClowdEnvironment
---------------------------------
Now that Clowder is running, we need to give it a `ClowdEnvironment` for Ingress to run inside. 

In a new terminal, run `oc new-project clowder-system`

Create the following as `clowd-environment.yaml`

``` yaml
---
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: ingress-env
spec:
  targetNamespace: clowder-system
  providers:
    web:
      port: 8000
      mode: operator
    metrics:
      port: 9000
      mode: operator
      path: "/metrics"
    kafka:
      namespace: default
      clusterName: crc-cluster
      mode: local
    db:
      image: "registry.redhat.io/rhel8/postgresql-12:1-36"
      mode: local
    logging:
      mode: none
    objectStore:
      mode: minio
      port: 9000
    inMemoryDb:
      mode: redis
  resourceDefaults:
    limits: 
      cpu: "500m"
      memory: "8192Mi"
    requests:
      cpu: "300m"
      memory: "1024Mi"

```

and then run `oc apply -f clowd-environment.yaml`

Once applied, check the terminal that is running the operator and make sure there aren't any errors. If you're unsure, you can check the `clowder-system` namespace. If you see issues with any types (like Kafka), run:
* `oc apply -f config/crd/bases/kafka.strimzi.io_kafkatopics.yaml`
* `oc apply -f config/crd/bases/kafka.strimzi.io_kafkas.yaml`

Before we add the ClowdApp, we need to port forward the minio port on your local machine with `oc port-forward svc/ingress-env-minio 9000`. Remember, in our example the operator is running on localhost. In order for our operator to talk to the minio service and perform bucket operations, we'll need to forward the port. 

Create the following file as `clowd-app.yaml` 


``` yaml
---
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: ingress
spec:
  envName: ingress-env 
  pods:
  - image: quay.io/cloudservices/insights-ingress-go-poc:5bcb3d14
    name: ingress
    livenessProbe:
      failureThreshold: 3
      httpGet:
        path: /api/ingress/v1/version
        port: 8000
        scheme: HTTP
      initialDelaySeconds: 10
      periodSeconds: 30
      successThreshold: 1
      timeoutSeconds: 1
    readinessProbe:
      failureThreshold: 3
      httpGet:
        path: /api/ingress/v1/version
        port: 8000
        scheme: HTTP
      initialDelaySeconds: 10
      periodSeconds: 30
      successThreshold: 1
      timeoutSeconds: 1
    env:
      - name: INGRESS_STAGEBUCKET
        value: ingress-uploads-perma
      - name: INGRESS_VALIDTOPICS
        value: advisor
      - name: OPENSHIFT_BUILD_COMMIT
        value: somestring
      - name: INGRESS_MINIODEV
        value: "true"
      - name: DEBUG
        value: "true"
    resources:
      limits:
        cpu: 300m
        memory: 8192Mi
      requests:
        cpu: 30m
        memory: 1024Mi
    web: true
  objectStore:
    - ingress-uploads-perma
  kafkaTopics:
    - replicas: 5
      partitions: 5
      topicName: advisor
    - replicas: 5
      partitions: 5
      topicName: platform.upload.advisor

```

Finally, `oc apply -f clowd-app.yaml`

If all works well you should see the operator terminal adding to the ingress namespace. Again, if you're unsure just checkout your crc in the `ingress` namespace. 

## Testing Ingress
-------------------

If you're interested in validating the new deployment, download [this tar to test it](https://gitlab.cee.redhat.com/insights-qe/iqe-core/-/blob/master/iqe/data/advisor_archives/security_low.tar.gz). 

Port forward the ingress port with `oc port-forward svc/ingress 8000`

Then run `curl -F "file=<YOUR DOWNLOAD LOCATION>/security_low.tar.gz;type=application/vnd.redhat.advisor.somefile+tgz" -H "x-rh-identity: eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjAwMDAwMSJ9fX0=" -H "x-rh-request_id: testtesttest" -v http://localhost:8000/api/ingress/v1/upload`

If you can see 
> We are completely uploaded and fine

It worked, and you are finished!
