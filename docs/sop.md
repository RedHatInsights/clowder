# Clowder Standard Operating Procedures (SOP)

This document provides standard operating procedures for managing, debugging, and releasing Clowder - the Red Hat Insights application configuration management operator for Kubernetes.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Debugging Procedures](#debugging-procedures)
3. [Release Procedures](#release-procedures)

---

## Architecture Overview

### High-Level Architecture

Clowder is a Kubernetes operator that manages application configuration and infrastructure dependencies for cloud-native applications. It consists of several key components:

#### Core Components

1. **Clowder Controller Manager**
   - Main operator process that watches for CRD changes
   - Reconciles ClowdApp and ClowdEnvironment resources
   - Manages application lifecycle and configuration generation

2. **Custom Resource Definitions (CRDs)**
   - `ClowdEnvironment`: Cluster-scoped resource defining infrastructure providers
   - `ClowdApp`: Namespace-scoped resource defining application specifications
   - `ClowdJobInvocation`: Resource for managing job executions

3. **Provider System**
   - Modular architecture supporting different infrastructure modes
   - Providers: Database, Kafka, Object Storage, Logging, Metrics, Web, etc.
   - Each provider supports multiple modes (local, operator, app-interface)

#### Data Flow

```
ClowdApp → Controller → Provider Logic → K8s Resources → Application Config
    ↓           ↓              ↓              ↓              ↓
 Spec      Reconcile     Infrastructure   Deployments   cdappconfig.json
```

#### Key Concepts

- **Environment Coupling**: ClowdApps are coupled to ClowdEnvironments via `envName`
- **Configuration Generation**: Apps receive standardized config via mounted secrets
- **Dependency Management**: Automatic service discovery and configuration injection
- **Provider Modes**: Flexible infrastructure provisioning strategies

### Deployment Architecture

Clowder is deployed via Operator Lifecycle Manager (OLM) with the following components:
- **OperatorGroup**: Defines operator scope and permissions
- **CatalogSource**: Points to operator bundle images
- **Subscription**: Manages operator installation and updates
- **ClusterServiceVersion**: Defines operator metadata and permissions

---

## Debugging Procedures

### Prerequisites

Before debugging Clowder issues, ensure you have:
- `kubectl` access to the target cluster
- Appropriate RBAC permissions to view Clowder resources
- Access to cluster logs and metrics

### Common Issues and Troubleshooting

#### 1. ClowdApp Not Deploying

**Symptoms:**
- ClowdApp resource exists but no deployments are created
- Application pods are not starting

**Debugging Steps:**

1. **Check ClowdApp Status:**
   ```bash
   kubectl get clowdapp <app-name> -n <namespace> -o yaml
   kubectl describe clowdapp <app-name> -n <namespace>
   ```

2. **Verify ClowdEnvironment:**
   ```bash
   kubectl get clowdenvironment <env-name> -o yaml
   kubectl describe clowdenvironment <env-name>
   ```

3. **Check Controller Logs:**
   ```bash
   kubectl logs -n clowder-system deployment/clowder-controller-manager -f
   ```

4. **Common Causes:**
   - Missing or invalid `envName` reference
   - ClowdEnvironment not ready
   - Missing dependencies in ClowdApp spec
   - Resource quota exceeded in target namespace

#### 2. Configuration Issues

**Symptoms:**
- Applications starting but failing to connect to services
- Missing configuration values in cdappconfig.json

**Debugging Steps:**

1. **Check Generated Configuration:**
   ```bash
   kubectl get secret <app-name> -n <namespace> -o jsonpath='{.data.cdappconfig\.json}' | base64 -d | jq
   ```

2. **Verify Provider Configuration:**
   ```bash
   kubectl get clowdenvironment <env-name> -o jsonpath='{.spec.providers}' | jq
   ```

3. **Check Provider Status:**
   ```bash
   kubectl describe clowdenvironment <env-name>
   ```

#### 3. Operator Not Responding

**Symptoms:**
- Changes to ClowdApp/ClowdEnvironment not being processed
- Controller manager pod crashing or restarting

**Debugging Steps:**

1. **Check Operator Health:**
   ```bash
   kubectl get pods -n clowder-system
   kubectl describe pod -n clowder-system -l app.kubernetes.io/name=clowder
   ```

2. **Review Controller Logs:**
   ```bash
   kubectl logs -n clowder-system deployment/clowder-controller-manager --previous
   ```

3. **Check Resource Usage:**
   ```bash
   kubectl top pods -n clowder-system
   kubectl describe node <node-name>
   ```

4. **Restart Controller:**
   ```bash
   kubectl rollout restart deployment/clowder-controller-manager -n clowder-system
   ```

#### 4. OLM Installation Issues

**Symptoms:**
- Clowder operator not installing via OLM
- CSV in failed state

**Debugging Steps:**

1. **Check OLM Resources:**
   ```bash
   kubectl get csv -n clowder-system
   kubectl get subscription -n clowder-system
   kubectl get catalogsource -n clowder-system
   ```

2. **Review CSV Status:**
   ```bash
   kubectl describe csv clowder.v<version> -n clowder-system
   ```

3. **Check OLM Operator Logs:**
   ```bash
   kubectl logs -n olm deployment/olm-operator
   kubectl logs -n olm deployment/catalog-operator
   ```

4. **Force Reinstall:**
   ```bash
   kubectl delete csv clowder.v<version> -n clowder-system
   kubectl delete subscription clowder -n clowder-system
   # Re-run saas-deploy job
   ```

#### 5. Performance Issues

**Symptoms:**
- Slow reconciliation times
- High memory/CPU usage
- Timeouts during resource creation

**Debugging Steps:**

1. **Monitor Resource Usage:**
   ```bash
   kubectl top pods -n clowder-system
   kubectl describe pod <controller-pod> -n clowder-system
   ```

2. **Check Reconciliation Metrics:**
   ```bash
   # Access Prometheus metrics endpoint
   kubectl port-forward -n clowder-system svc/clowder-controller-manager-metrics-service 8080:8080
   curl http://localhost:8080/metrics | grep controller_runtime
   ```

3. **Review Controller Configuration:**
   ```bash
   kubectl get configmap clowder-config -n clowder-system -o yaml
   ```

### Log Analysis

#### Controller Manager Logs

Key log patterns to look for:

- **Reconciliation Errors:**
  ```
  ERROR   controller-runtime.manager.controller.clowdapp   Reconciler error
  ```

- **Provider Failures:**
  ```
  ERROR   providers.<provider-name>   Failed to reconcile provider
  ```

- **Resource Creation Issues:**
  ```
  ERROR   controllers.ClowdApp   unable to create deployment
  ```

#### Useful Log Commands

```bash
# Follow controller logs with filtering
kubectl logs -n clowder-system deployment/clowder-controller-manager -f | grep ERROR

# Get logs for specific ClowdApp reconciliation
kubectl logs -n clowder-system deployment/clowder-controller-manager | grep "clowdapp/<app-name>"

# Export logs for analysis
kubectl logs -n clowder-system deployment/clowder-controller-manager --since=1h > clowder-logs.txt
```

### Emergency Procedures

#### Complete Operator Reset

**⚠️ WARNING: This will cause downtime for all managed applications**

1. **Scale down controller:**
   ```bash
   kubectl scale deployment clowder-controller-manager --replicas=0 -n clowder-system
   ```

2. **Clean up stuck resources:**
   ```bash
   kubectl patch clowdapp <app-name> -n <namespace> --type merge -p '{"metadata":{"finalizers":[]}}'
   ```

3. **Restart operator:**
   ```bash
   kubectl scale deployment clowder-controller-manager --replicas=1 -n clowder-system
   ```

#### Cluster-wide Resource Cleanup

```bash
# List all Clowder resources
kubectl get clowdapps --all-namespaces
kubectl get clowdenvironments

# Force delete stuck resources (use with caution)
kubectl patch clowdenvironment <env-name> --type merge -p '{"metadata":{"finalizers":[]}}'
```

---

## Release Procedures

### Release Types

Clowder follows semantic versioning (SemVer) with the following release types:

- **Patch Release (x.y.Z)**: Bug fixes, security patches, minor improvements
- **Minor Release (x.Y.z)**: New features, API additions, backward-compatible changes
- **Major Release (X.y.z)**: Breaking changes, API modifications, major architectural updates

### Pre-Release Checklist

Before initiating a release, ensure:

- [ ] All planned features/fixes are merged to `main` branch
- [ ] CI/CD pipeline is passing on `main` branch
- [ ] E2E tests are passing in staging environment
- [ ] Documentation is updated for new features
- [ ] Breaking changes are documented in migration guide
- [ ] Security scan results are reviewed and approved
- [ ] Performance regression tests are passing

### Release Process

#### 1. Prepare Release Branch

```bash
# Create release branch from main
git checkout main
git pull origin main
git checkout -b release/v<version>

# Update version in relevant files
# - Update VERSION file
# - Update operator bundle manifests
# - Update documentation references
```

#### 2. Generate Release Notes

```bash
# Generate changelog since last release
git log --oneline --no-merges v<previous-version>..HEAD

# Create release notes including:
# - New features and enhancements
# - Bug fixes
# - Breaking changes
# - Known issues
# - Upgrade instructions
```

#### 3. Build and Test Release Candidate

```bash
# Build release candidate images
make docker-build IMG=quay.io/cloudservices/clowder:v<version>-rc1
make bundle-build BUNDLE_IMG=quay.io/cloudservices/clowder-bundle:v<version>-rc1

# Push release candidate images
make docker-push IMG=quay.io/cloudservices/clowder:v<version>-rc1
make bundle-push BUNDLE_IMG=quay.io/cloudservices/clowder-bundle:v<version>-rc1

# Deploy to staging environment for testing
# Run comprehensive test suite
make test-e2e
```

#### 4. Create Release Tag

```bash
# Tag the release
git tag -a v<version> -m "Release v<version>"
git push origin v<version>

# Create GitHub release
# - Upload release artifacts
# - Include release notes
# - Mark as pre-release if RC
```

#### 5. Build Production Images

```bash
# Build final release images
make docker-build IMG=quay.io/cloudservices/clowder:v<version>
make bundle-build BUNDLE_IMG=quay.io/cloudservices/clowder-bundle:v<version>
make catalog-build CATALOG_IMG=quay.io/cloudservices/clowder-catalog:v<version>

# Push production images
make docker-push IMG=quay.io/cloudservices/clowder:v<version>
make bundle-push BUNDLE_IMG=quay.io/cloudservices/clowder-bundle:v<version>
make catalog-push CATALOG_IMG=quay.io/cloudservices/clowder-catalog:v<version>
```

#### 6. Deploy to Staging

```bash
# Update staging environment
# - Update CatalogSource with new catalog image
# - Monitor deployment health
# - Run smoke tests
# - Validate application functionality
```

#### 7. Production Deployment

**⚠️ Production deployments require additional approvals and coordination**

1. **Create App-Interface MR:**
   ```yaml
   # Update saas file with new image references
   resourceTemplates:
   - name: clowder-catalog
     targets:
     - namespace: clowder-system
       ref: <commit-hash>  # Update this
   ```

2. **Coordinate Deployment:**
   - Schedule deployment window
   - Notify stakeholders
   - Prepare rollback plan
   - Monitor cluster capacity

3. **Execute Deployment:**
   ```bash
   # Merge app-interface MR
   # Monitor OLM deployment
   kubectl get csv -n clowder-system -w
   
   # Verify operator health
   kubectl get pods -n clowder-system
   kubectl logs -n clowder-system deployment/clowder-controller-manager
   ```

4. **Post-Deployment Validation:**
   - Verify all ClowdApps are reconciling
   - Check application configurations
   - Monitor error rates and performance
   - Validate new features (if applicable)

### Rollback Procedures

#### Emergency Rollback

If critical issues are discovered post-deployment:

1. **Immediate Rollback:**
   ```bash
   # Revert to previous catalog image
   kubectl patch catalogsource clowder-catalog -n clowder-system \
     --type merge -p '{"spec":{"image":"quay.io/cloudservices/clowder-catalog:v<previous-version>"}}'
   
   # Force CSV recreation
   kubectl delete csv clowder.v<current-version> -n clowder-system
   ```

2. **Monitor Rollback:**
   ```bash
   # Watch operator rollback
   kubectl get csv -n clowder-system -w
   kubectl get pods -n clowder-system -w
   ```

3. **Validate Rollback:**
   - Verify operator is running previous version
   - Check ClowdApp reconciliation
   - Validate application functionality

#### Planned Rollback

For planned rollbacks (e.g., during maintenance):

1. Create app-interface MR reverting image references
2. Follow standard deployment process
3. Communicate changes to stakeholders

### Post-Release Activities

#### 1. Update Documentation

- [ ] Update API reference documentation
- [ ] Refresh user guides and tutorials
- [ ] Update migration guides
- [ ] Publish release blog post (if major release)

#### 2. Monitor Release Health

```bash
# Monitor key metrics for 24-48 hours
# - Reconciliation success rate
# - Error rates in logs
# - Resource utilization
# - Application deployment success

# Set up alerts for:
# - Controller restart loops
# - High error rates
# - Performance degradation
```

#### 3. Gather Feedback

- Monitor support channels for issues
- Review user feedback and bug reports
- Track adoption metrics
- Plan hotfix releases if needed

### Hotfix Release Process

For critical bug fixes that cannot wait for the next regular release:

1. **Create Hotfix Branch:**
   ```bash
   git checkout v<last-release>
   git checkout -b hotfix/v<version>-hotfix1
   ```

2. **Apply Minimal Fix:**
   - Cherry-pick specific commits
   - Avoid unnecessary changes
   - Update version to patch level

3. **Fast-Track Testing:**
   - Focus on regression testing
   - Validate fix effectiveness
   - Skip non-critical test suites

4. **Expedited Deployment:**
   - Follow abbreviated release process
   - Coordinate with stakeholders
   - Monitor closely post-deployment

### Release Metrics and KPIs

Track the following metrics for release quality:

- **Lead Time**: Time from feature complete to production
- **Deployment Frequency**: How often releases are deployed
- **Mean Time to Recovery (MTTR)**: Time to recover from failures
- **Change Failure Rate**: Percentage of releases causing issues
- **Rollback Rate**: Percentage of releases requiring rollback

### Release Calendar

Maintain a regular release schedule:

- **Major Releases**: Quarterly (every 3 months)
- **Minor Releases**: Monthly
- **Patch Releases**: As needed (typically bi-weekly)
- **Hotfix Releases**: Emergency only

### Communication Plan

For each release:

1. **Pre-Release (1 week before):**
   - Announce upcoming release
   - Share release notes draft
   - Coordinate with dependent teams

2. **Release Day:**
   - Announce release completion
   - Share final release notes
   - Provide support contact information

3. **Post-Release (1 week after):**
   - Share adoption metrics
   - Address any issues or feedback
   - Plan next release cycle
