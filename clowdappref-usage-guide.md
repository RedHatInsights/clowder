# ClowdAppRef Usage Guide

## Overview

`ClowdAppRef` is a Clowder Custom Resource Definition (CRD) that allows you to reference and consume services running on external clusters or environments. This enables your ClowdApps to depend on services that exist outside your current cluster while maintaining the same configuration injection pattern that Clowder provides for local dependencies.

## Key Benefits

1. **Multi-cluster Dependencies**: Reference services across different clusters seamlessly
2. **Consistent Configuration**: External services appear in app config just like local ClowdApp dependencies
3. **Service Discovery**: Automatic endpoint configuration for consuming applications
4. **Environment Flexibility**: Same ClowdApp works across different environments with different external service configurations
5. **Gradual Migration**: Move services between clusters without changing consuming applications

## How It Works

When you create a ClowdAppRef and reference it in a ClowdApp's dependencies:

1. **ClowdAppRef** defines external services with their connection details
2. **ClowdApp** lists the ClowdAppRef in its `dependencies` or `optionalDependencies`
3. **Clowder Controller** processes the dependency and injects endpoint configuration
4. **Application Config** receives external service endpoints in the same format as local dependencies

## ClowdAppRef Structure

### Required Fields

- `envName`: The ClowdEnvironment this ClowdAppRef belongs to
- `deployments`: List of external service deployments

### Deployment Configuration

Each deployment in the ClowdAppRef can specify:

```yaml
- name: service-name                    # Required: Service identifier
  hostname: service.external.com        # Required: External hostname
  port: 8080                           # Optional: HTTP port (default: 8000)
  tlsPort: 8443                        # Optional: HTTPS port (default: 8443)
  privatePort: 10000                   # Optional: Private HTTP port (default: 10000)
  tlsPrivatePort: 10443               # Optional: Private HTTPS port (default: 10443)
  web: true                           # Optional: Has public web service
  webServices:                        # Optional: Web service configuration
    public:
      enabled: true
    private:
      enabled: true
  apiPath: "/api/service/"            # Optional: Single API path (deprecated)
  apiPaths:                           # Optional: Multiple API paths
    - "/api/service/v1/"
    - "/api/service/v2/"
```

### Optional Fields

- `disabled`: Turn off the ClowdAppRef
- `remoteCluster`: Metadata about the external cluster

## Configuration Injection

When a ClowdApp depends on a ClowdAppRef, the external service endpoints are injected into the app's configuration secret (`cdappconfig.json`) under the `endpoints` array:

```json
{
  "endpoints": [
    {
      "app": "external-services",
      "name": "user-management",
      "hostname": "users.production-cluster.company.com",
      "port": 8080,
      "tlsPort": 8443,
      "privatePort": 10000,
      "tlsPrivatePort": 10443
    },
    {
      "app": "external-services", 
      "name": "payment-processor",
      "hostname": "payments.production-cluster.company.com",
      "port": 8080,
      "tlsPort": 8443,
      "privatePort": 10000,
      "tlsPrivatePort": 10443
    }
  ]
}
```

## Usage Patterns

### 1. Simple External Service Reference

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdAppRef
metadata:
  name: shared-services
  namespace: my-namespace
spec:
  envName: my-environment
  deployments:
  - name: auth-service
    hostname: auth.shared-cluster.com
    port: 8080
    tlsPort: 8443
    apiPath: "/api/auth/"
```

### 2. Multiple Services with Different Configurations

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdAppRef
metadata:
  name: backend-services
  namespace: my-namespace  
spec:
  envName: my-environment
  deployments:
  # Public-facing API service
  - name: api-gateway
    hostname: api.company.com
    port: 80
    tlsPort: 443
    web: true
    webServices:
      public:
        enabled: true
    apiPaths:
    - "/api/v1/"
    - "/api/v2/"
  
  # Internal service
  - name: internal-processor  
    hostname: processor.internal.company.com
    privatePort: 9000
    tlsPrivatePort: 9443
    web: false
    webServices:
      private:
        enabled: true
```

### 3. Environment-Specific External Services

**Development Environment:**
```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdAppRef
metadata:
  name: external-apis
  namespace: dev-namespace
spec:
  envName: development
  deployments:
  - name: payment-service
    hostname: payments-dev.company.com
    port: 8080
```

**Production Environment:**
```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdAppRef
metadata:
  name: external-apis
  namespace: prod-namespace
spec:
  envName: production
  deployments:
  - name: payment-service
    hostname: payments.company.com
    port: 443
    tlsPort: 443
```

## Consuming ClowdAppRefs in ClowdApps

### Basic Dependency

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: my-application
spec:
  envName: my-environment
  deployments:
  - name: worker
    podSpec:
      image: my-org/worker:latest
  
  dependencies:
    - external-apis  # ClowdAppRef name
```

### Mixed Dependencies

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdApp
metadata:
  name: complex-app
spec:
  envName: my-environment
  deployments:
  - name: api
    podSpec:
      image: my-org/api:latest
  
  dependencies:
    - external-apis     # ClowdAppRef
    - local-database    # Another ClowdApp
    - shared-cache      # Another ClowdApp
  
  optionalDependencies:
    - analytics-service # Optional ClowdAppRef
```

## Best Practices

### 1. Naming Conventions
- Use descriptive names for ClowdAppRefs (e.g., `shared-services`, `external-apis`)
- Use consistent deployment names across environments
- Include environment context in hostnames when appropriate

### 2. Port Configuration
- Always specify TLS ports for production services
- Use standard ports (80/443) for public services
- Use high-numbered ports (10000+) for private services
- Be consistent with port usage across similar services

### 3. API Path Management
- Use `apiPaths` instead of deprecated `apiPath`
- Include version information in API paths
- Ensure paths end with `/` for consistency

### 4. Environment Separation
- Create separate ClowdAppRefs for different environments
- Use environment-specific hostnames
- Maintain same ClowdAppRef names across environments for consistency

### 5. Service Organization
- Group related services in a single ClowdAppRef
- Separate by security domains (public vs private)
- Consider network topology when grouping services

## Troubleshooting

### Common Issues

1. **ClowdAppRef not found**: Ensure the ClowdAppRef exists in the same namespace as the consuming ClowdApp
2. **Missing endpoints**: Check that the ClowdAppRef name matches exactly in the ClowdApp dependencies
3. **Configuration not updating**: Verify the ClowdApp controller has processed the dependency changes
4. **External service unreachable**: Validate hostnames and ports are correct and accessible from the cluster

### Debugging Configuration

Check the generated configuration:
```bash
# Get the app's configuration secret
kubectl get secret <app-name> -o jsonpath='{.data.cdappconfig\.json}' | base64 -d | jq

# Verify ClowdAppRef status
kubectl get clowdappref <clowdappref-name> -o yaml

# Check ClowdApp status and dependencies
kubectl get clowdapp <app-name> -o yaml
```

## Migration Scenarios

### Moving Services Between Clusters

1. **Create ClowdAppRef** for the service in the new location
2. **Update consuming ClowdApps** to reference the ClowdAppRef instead of direct ClowdApp dependency
3. **Migrate the service** to the external cluster
4. **Update ClowdAppRef hostname** to point to the new location
5. **Remove old ClowdApp** from the original cluster

This approach allows for zero-downtime migrations and gradual rollouts.

## Security Considerations

- External services should use TLS for production traffic
- Consider network policies to restrict access to external services
- Validate that external hostnames are accessible from your cluster
- Use private ports for internal-only communications
- Implement proper authentication/authorization for external service access 