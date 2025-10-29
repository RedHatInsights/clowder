import os
import json
import subprocess
import time

import pytest


def _run(cmd: list[str]) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=False)

def _get_cdappconfig_json(namespace: str, app_name: str) -> dict:
    cp = _run([
        "oc",
        "get",
        "secret",
        app_name,
        "-n",
        namespace,
        "-o",
        "json",
    ])
    assert cp.returncode == 0, f"Failed to get secret {app_name} in ns {namespace}: {cp.stderr}"
    secret = json.loads(cp.stdout)
    raw_b64 = secret.get("data", {}).get("cdappconfig.json", "")
    assert raw_b64, "cdappconfig.json not found in secret data"
    from base64 import b64decode
    content = b64decode(raw_b64).decode("utf-8")
    return json.loads(content)

def test_cdappconfig_secret_exists():
    namespace = os.environ.get("TEST_NS", "clowder-e2e")
    app_name = os.environ.get("CLOWDAPP_NAME", "puptoo")

    # Wait up to 5 minutes for Secret to appear
    secret_name = app_name  # convention: secret name equals app name
    for _ in range(60):
        cp = _run(["oc", "get", "secret", secret_name, "-n", namespace, "-o", "jsonpath={.data.cdappconfig\\.json}"])
        if cp.returncode == 0 and cp.stdout.strip():
            break
        time.sleep(5)
    else:
        pytest.fail(f"cdappconfig secret {secret_name} not found or missing data in ns {namespace}")

    # Validate it parses as JSON via helper
    _ = _get_cdappconfig_json(namespace, app_name)

def test_cdappconfig_content_exact_match():
    namespace = os.environ.get("TEST_NS", "clowder-e2e")
    app_name = os.environ.get("CLOWDAPP_NAME", "puptoo")

    expected_str = (
        '{"endpoints":[{"apiPath":"/api/puptoo-processor/","apiPaths":["/api/puptoo-processor/"],'
        f'"app":"puptoo","h2cPort":0,"h2cTLSPort":0,"hostname":"puptoo-processor.{namespace}.svc",'
        '"name":"processor","port":8000,"tlsPort":0}],"hashCache":"584847c5d012b85e0b73ea34f93678ef4c21a9dc1312ae29f545f6412d03ee28e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",'
        '"logging":{"cloudwatch":{"accessKeyId":"","logGroup":"","region":"","secretAccessKey":""},"type":"null"},'
        '"metadata":{"deployments":[{"image":"quay.io/psav/clowder-hello","name":"processor"}],"envName":"test-basic-app","name":"puptoo"},'
        '"metricsPath":"/metrics","metricsPort":9000,'
        f'"privateEndpoints":[{"app":"puptoo","h2cPort":0,"h2cTLSPort":0,"hostname":"puptoo-processor.{namespace}.svc","name":"processor","port":10000,"tlsPort":0}],'
        '"privatePort":10000,"publicPort":8000,"webPort":8000}'
    )
    expected = json.loads(expected_str)
    actual = _get_cdappconfig_json(namespace, app_name)
    assert actual == expected, (
        "cdappconfig mismatch.\nExpected: "
        + json.dumps(expected, sort_keys=True)
        + "\nActual: "
        + json.dumps(actual, sort_keys=True)
    )
