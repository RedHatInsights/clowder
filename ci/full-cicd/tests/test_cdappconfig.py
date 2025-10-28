import os
import json
import subprocess
import time

import pytest


def _run(cmd: list[str]) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=False)


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

    # Basic sanity: can we decode the JSON content?
    # Use oc to decode base64
    cp_json = _run(["sh", "-lc", f"oc get secret {secret_name} -n {namespace} -o jsonpath='{{.data.cdappconfig\\.json}}' | base64 -d"])
    assert cp_json.returncode == 0, cp_json.stderr
    assert cp_json.stdout.strip(), "cdappconfig.json content is empty"

    # Validate it parses as JSON
    json.loads(cp_json.stdout)
