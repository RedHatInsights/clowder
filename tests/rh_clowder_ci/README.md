# Red Hat Clowder E2E Test Suite

This package contains end-to-end tests for the Clowder operator.

## Installation

This package is designed for local installation only (not published to PyPI):

```bash
cd tests/rh_clowder_ci
pip install -e .
```

## Running Tests

You must be logged into an OpenShift cluster before running tests:

```bash
# Set environment variables (optional)
export TEST_NS=clowder-e2e-test

# Run tests
pytest
```

## Environment Variables

- `TEST_NS`: Namespace to deploy test resources into (default: `clowder-e2e`)
- `CLOWDAPP_NAME`: Name of the ClowdApp to test (default: `puptoo`)
- `WAIT_TIMEOUT`: Timeout for resource readiness checks (default: `5m`)
- `RESOURCES_PATH`: Path to test resources template (optional, uses bundled template by default)

## What Gets Tested

The E2E tests:
1. Deploy a ClowdEnvironment and ClowdApp using OpenShift templates
2. Wait for all resources to be ready
3. Verify the `cdappconfig.json` secret is created with correct content
4. Clean up all created resources after tests complete

## Package Structure

- `rh_clowder_ci/tests/`: Test modules
- `rh_clowder_ci/resources/`: OpenShift templates and test resources
