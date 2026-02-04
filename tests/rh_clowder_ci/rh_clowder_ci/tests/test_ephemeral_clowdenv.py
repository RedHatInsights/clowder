from bonfire.openshift import wait_for_all_resources


def test_clowdenvironment(deploy_test_resources, namespace):
    deploy_test_resources("eno-clowdenvironment.yaml")
    wait_for_all_resources(namespace)
