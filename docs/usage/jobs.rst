Job and CronJob Support
=======================

CronJobs are currently enabled as part of the ClowdApp spec. The ``jobs`` field
contains a list of all currently defined jobs. The spec for a job is documented
in the `Clowder API reference`_. 

Current Limitations
-------------------
There is currently only support for CronJobs. CronJobs are specific 
implementations of Jobs, but use ``schedule`` as their trigger mechanism.
Currently, ClowdApps only support jobs that have a schedule defined. If your
defined job does not include the schedule field, the ClowdApp will no longer
be valid. The devprod team is actively working on support for Jobs and the 
mechanism to trigger them in the app-sre clusters.


Examples
--------
The following is a shortened excerpt from Advisor for demonstration. 

.. code-block:: yaml

  apiVersion: v1
  kind: Template
  metadata:
    name: advisor-backend
  objects:
  - apiVersion: cloud.redhat.com/v1alpha1
    kind: ClowdApp
    metadata:
      name: advisor-backend
    spec:
      envName: ${ENV_NAME}
      jobs:
      - name: content-server
        schedule: 50 11 * * *
        podSpec:
          image: quay.io/cloudservices/content-server
          command:
            - /bin/bash
            - -c
            - |
              whoami &> /dev/null
          resources:
            limits:
              cpu: 500m
              memory: 512Mi
            requests:
              cpu: 200m
              memory: 256Mi

.. _Clowder API reference: https://redhatinsights.github.io/clowder/api_reference.html#k8s-api-cloud-redhat-com-clowder-v2-apis-cloud-redhat-com-v1alpha1-job
.. vim: tw=80 spell spelllang=en
