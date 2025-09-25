# Sidecar Memory Configuration Tests

This test suite validates the memory configuration feature for otel-collector sidecars introduced in PR #1358.

## Overview

The memory configuration feature allows users to configure memory requests and limits for otel-collector sidecars at both the ClowdEnvironment and ClowdApp levels, with a clear priority hierarchy:

1. **ClowdApp level** (highest priority) - `sidecars[].memoryRequest` and `sidecars[].memoryLimit`
2. **ClowdEnvironment level** - `providers.sidecars.otelCollector.memoryRequest` and `providers.sidecars.otelCollector.memoryLimit`
3. **Default values** (lowest priority) - `512Mi` request, `1024Mi` limit

## Test Cases

### 1. Default Memory Settings (`01-default-memory.yaml`)
- **Purpose**: Verify that default memory settings are applied when no custom configuration is provided
- **Expected**: `512Mi` memory request, `1024Mi` memory limit
- **Validates**: Backward compatibility and default behavior

### 2. Environment-Level Memory Configuration (`02-environment-memory.yaml`)
- **Purpose**: Test memory configuration at the ClowdEnvironment level
- **Configuration**: `256Mi` request, `2048Mi` limit at environment level
- **Expected**: All apps in the environment use the environment-level settings
- **Validates**: Environment-level configuration inheritance

### 3. App-Level Memory Configuration (`03-app-memory.yaml`)
- **Purpose**: Test memory configuration at the ClowdApp level
- **Configuration**: `128Mi` request, `4096Mi` limit at app level
- **Expected**: App-specific memory settings override defaults
- **Validates**: App-level configuration override

### 4. Priority Testing (`04-priority-test.yaml`)
- **Purpose**: Verify the configuration priority hierarchy
- **Test Scenarios**:
  - **Partial override**: App overrides only `memoryRequest`, uses environment `memoryLimit`
  - **Full override**: App overrides both `memoryRequest` and `memoryLimit`
  - **No override**: Uses environment-level settings
- **Validates**: Priority: ClowdApp > ClowdEnvironment > defaults

### 5. CronJob Memory Configuration (`05-cronjob-memory.yaml`)
- **Purpose**: Test memory configuration for CronJob sidecars
- **Configuration**: `128Mi` request, `1536Mi` limit for cronjob sidecar
- **Expected**: CronJob sidecar uses app-level memory settings
- **Validates**: Memory configuration works for both Deployments and CronJobs

## Resource Verification

Each test verifies that the generated Kubernetes resources (Deployments and CronJobs) have the correct memory settings in their `initContainers` section:

```yaml
initContainers:
- name: otel-collector
  resources:
    limits:
      memory: "<expected-limit>"
    requests:
      memory: "<expected-request>"
```

## Running the Tests

To run this specific test suite:

```bash
kubectl kuttl test tests/kuttl/test-sidecar-memory-config/
```

To run all kuttl tests:

```bash
make test-kuttl
```

## Implementation Details

The memory configuration is implemented through:

- **API Fields**: `memoryRequest` and `memoryLimit` in both `Sidecar` and `OtelCollectorConfig` structs
- **Helper Functions**: `GetOtelCollectorMemoryRequest()` and `GetOtelCollectorMemoryLimit()` in the sidecar provider
- **Resource Application**: Memory settings are applied to the otel-collector initContainer in both Deployments and CronJobs

## Related Files

- **API Definitions**: `apis/cloud.redhat.com/v1alpha1/clowdapp_types.go`, `apis/cloud.redhat.com/v1alpha1/clowdenvironment_types.go`
- **Implementation**: `controllers/cloud.redhat.com/providers/sidecar/provider.go`, `controllers/cloud.redhat.com/providers/sidecar/default.go`
- **Unit Tests**: `controllers/cloud.redhat.com/providers/sidecar/provider_test.go`

