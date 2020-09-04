# Whipporwill :bird: - Insights Platform Operator

An operator to deploy and operate cloud.redhat.com resources for Openshift.

## Design

[Design docs](https://gitlab.cee.redhat.com/klape/operator-design/)

## Dependencies

- [Operator SDK](https://github.com/operator-framework/operator-sdk/releases)
- [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder/releases)
- [kustomize](https://github.com/kubernetes-sigs/kustomize/releases)
- Either Codeready Containers or a remote cluster where you have access to
  create CRDs.

## Running

- `make install` will deploy the CRDs to the cluster configured in your kubeconfig.
- `make run` will build the binary and locally run the binary, connecting the
  manager to the Openshift cluster configured in your kubeconfig.
- `make deploy` will try to run the manager in the cluster configured in your
  kubeconfig.  You likely need to push the image to an image stream local to
  your target namespace.

## Testing

The tests rely on the test environment set up by controller-runtime.  This
enables the operator to get initialized against a control pane just like it
would against a real Openshift cluster.

While the tests do not rely on any additional testing frameworks (e.g. Ginkgo),
you do need to download
[kubebuilder](https://github.com/kubernetes-sigs/kubebuilder/releases) in order
to set up the control plane used
by the controller-runtime test environment.

Run the tests:

```
$ KUBEBUILDER_ASSETS=$PWD/kubebuilder go test ./controllers/...
ok      cloud.redhat.com/whippoorwill/v2/controllers    9.626s
```
