Testing
=======

Clowder testing utilises two main testing techniques:

* Unit tests - small fast tests of individual functions
* Kuttl/E2E tests - E2E tests run in a real cluster

The development of tests for these two categories, the sections below detail
some of the development flows for writing tests.

Types of tests
++++++++++++++

Unit tests
----------

The ``controllers/cloud.redhat.com/suite_test.go`` is the test file for most of
the top level functions in Clowder. Some providers also have their own test
files to assert specific functionality. This suite does have an etcd process
initiated as part of the test run, but does not have any operators running as
you would expect on a cluster. For example, if a Deployment resource is created
and applied, a Pod resource will NOT be created as it otherwise would be. If
specific functionality is expected to be tested like this, the Kuttl/E2E tests
should be used.

Kuttl/E2E tests
---------------

The E2E tests make use of the Kuttl suite to test the application and
subsequent result of applying certain resources in a cluster which is running
the Clowder operator. Kuttl applies certain resources, and then asserts that
the resulting resources match those specified. It is suggested to look at the
many examples in the ``bundle/tests/scoredcard/kuttl`` directory. They are
generally broken down into the following structure. 

.. code-block:: text

    kuttl/
    └── test-name/
        ├── 00-install.yaml
        ├── 01-pods.yaml
        ├── 01-assert.yaml
        ├── 02-json-asserts.yaml
        └── 03-delete.yaml

The numerals infront of the test steps define the order Kuttle will invoke
them. The only specially named files are the ``*-assert`` files, which are
always run last. Sometimes the ordering is forced, e.g. you will usually see
the ``delete`` files in a separate step at the end to clean up as best it can.

``00-install.yaml``
*******************

Kuttl usually creates a random namespace for a particular test, but in the
Clowder E2E test suite, the name is required for certain assertions and Kuttl
lacks the means for the E2E suite to reliably retrieve it. The
``00-install.yaml`` file usually contains a namespace definition that houses
the test input and output resources.

``01-pods.yaml``
****************

Called ``pods`` because it will usually contain the definitions that will lead
to pods being created.

``01-assert.yaml``
******************

The resources in this file ill be compared to the ones in the cluster. Kuttl
will wait for a period of time until the resources in the cluster match the
resources in the file. If they do not match when the timeout occurs, the test
is said to have failed.

``02-json-asserts.yaml``
************************

This is a hack as when Kuttl was first introduced it could not run commands as
tests, only as steps in preparing environments. As the inability to complete a
command would halt the test with a failure, the ``json-asserts`` files are
often used to assert that certain pieces of the JSON (cdappconfig.json) secret
are correct. As these are base64 encoded and contain a blob of data, Kuttl has
no way of matching the resource, so we use the ``jq`` command to assert
instead.

``03-delete.yaml``
******************

Deletions of the namespace and other resources allow the minikube environment
to be kept as clean as possible during the test run. Leaving pods only
increases resource usage unnecessarily.

Running tests
+++++++++++++

To invoke either the unit tests, or the Kuttl tests, the kubebuilder assets are
required to either be on path, or an environment variable needs to be set to
point to them. The example below shows how to run the unit tests by setting the
environment variable.

.. code-block:: shell

    KUBEBUILDER_ASSETS=~/kubebuilder_2.3.1_linux_amd64/bin/ make test

Running the Kuttl tests requires a cluster to be present. It is possible to run
the Kuttl tests with a simple mocked backplane, but with the complex
integration between multiple operators, the Kuttl tests in Clowder are run
against minikube. With a minikube instance installed and configure as the
default for ``kubectl``, the following command will run **all** the e2e tests.

.. code-block:: shell

    KUBEBUILDER_ASSETS=~/kubebuilder_2.3.1_linux_amd64/bin/ \
        kubectl kuttl test \
        --config bundle/tests/scorecard/kuttl/kuttl-test.yaml \
        --manifest-dir config/crd/bases/ \
        --manifest-dir config/crd/static/ \
        bundle/tests/scorecard/kuttl/


Single tests can be targetted using the ``--test`` command line flag and using
the name of the directory of the test to be run.
