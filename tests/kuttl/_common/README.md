# KUTTL Test Event Collection

This directory contains common utilities for KUTTL tests, including automatic Kubernetes event collection on test failures.

## How It Works

When KUTTL tests fail, Kubernetes events are automatically collected and saved to help with debugging.

### For TestAssert Failures (Standard K8s Resource Assertions)

All TestAssert files now include a KUTTL collector that runs when assertions fail:

```yaml
---
apiVersion: kuttl.dev/v1beta1
kind: TestAssert
collectors:
- type: command
  command: bash ../_common/collect-events.sh
  timeout: 10
---
# Kubernetes resource assertions follow...
```

When any assertion fails, the collector script runs and saves events from all namespaces defined in the test's `00-install.yaml` file. Events are saved to:
`artifacts/kuttl/<test-name>/events-<namespace>.txt` (one file per namespace)

### For TestStep Failures (JSON Assert Scripts)

**Files affected**: `*-json-asserts.yaml` and `*.sh` files

JSON assert tests use shell scripts with built-in error handling:

1. Each test has a shell script (e.g., `02-json-asserts.sh`)
2. The script sources `error-handler.sh` which sets up a trap
3. When any command fails, the trap automatically collects events from all namespaces
4. Events are saved to: `artifacts/kuttl/<test-name>/events-<namespace>.txt` (one file per namespace)

## Scripts

### error-handler.sh
Common error handling library that all test scripts source. Provides:
- `setup_error_handling(test_name)` function - takes only the test name as parameter
- Automatic event collection via bash traps from all test namespaces
- Uses `$NAMESPACE` environment variable (set by KUTTL) when `00-install.yaml` is not found
- Error exit handling

### collect-events.sh
Event collector script called by KUTTL collectors. Automatically discovers all namespaces defined in the test's `00-install.yaml` file and collects events from each. When `00-install.yaml` is not found, uses the `$NAMESPACE` environment variable set by KUTTL. Handles cases where namespaces don't exist yet gracefully. Can also be called manually for debugging.

## Event Collection Locations

All events are saved to:
```
artifacts/
  kuttl/
    <test-name>/
      events-<namespace>.txt
```

For example:
```
artifacts/
  kuttl/
    test-basic-app/
      events-test-basic-app.txt
      events-test-basic-app-secret.txt
    test-kafka-managed/
      events-test-kafka-managed.txt
    test-shared-elasticache/
      events-test-shared-elasticache.txt
      events-test-shared-elasticache-ns2.txt
```

**Note**: The scripts automatically discover all namespaces by parsing the `00-install.yaml` file in each test directory. If a namespace doesn't exist yet (e.g., test failed before namespace creation), event collection for that namespace is skipped gracefully without error.
