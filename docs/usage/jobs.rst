Job and CronJob Support
=======================

Jobs and CronJobs are currently enabled as part of the ClowdApp spec. The
``jobs`` field contains a list of all currently defined jobs. The spec for a 
job is documented in the `Clowder API reference`_. 

Jobs and CronJobs are split by a ``schedule`` field inside your job. If the job
has a ``schedule``, it is assumed to be a CronJob. If not, Clowder runs your 
job as a standard Job resource. Note that Jobs run as soon as they are applied. 

Jobs that need to run as soon as your deployment is ready are marked by the
``oneshot`` field. These jobs run once and are finished forever.
You cannot have a ``schedule`` and ``oneshot`` enabled. 


Triggering Jobs via App-interface
---------------------------------
Coming Soon!

Triggering Jobs in Ephemeral Environments
-----------------------------------------
Coming Soon!


Ordering Jobs
-------------
Coming Soon!

Examples
--------

.. code-block:: yaml

    ---
    apiVersion: v1
    kind: Template
    metadata:
      name: jobs
    objects:
    - apiVersion: cloud.redhat.com/v1alpha1
      kind: ClowdApp
      metadata:
        name: job
      spec:
        envName: env-debugger
        jobs:
          - name: no-prio
            schedule: "*/20 * * * *"
            podSpec:
              name: hello
              image: busybox
              imagePullPolicy: IfNotPresent
              args:
              - /bin/sh
              - -c
              - date; echo Hello from the Kubernetes cluster
          - name: high-prio
            podSpec:
              name: hello
              image: busybox
              imagePullPolicy: IfNotPresent
              oneshot: true
              args:
              - /bin/sh
              - -c
              - date; echo Hello from the Kubernetes cluster
          - name: low-prio
            podSpec:
              name: hello
              image: busybox
              imagePullPolicy: IfNotPresent
              oneshot: true
              args:
              - /bin/sh
              - -c
              - date; echo Hello from the Kubernetes cluster
    parameters:
      - name: LOG_LEVEL
        value: 'INFO'

.. _Clowder API reference: https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-job
.. vim: tw=80 spell spelllang=en
