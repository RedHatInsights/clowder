# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Clowder is a Kubernetes operator designed to make it easy to deploy applications running on the cloud.redhat.com platform in production, testing and local development environments. It's a Go-based project using the Kubebuilder framework and controller-runtime library.

### What Clowder Does

Clowder takes application definitions (ClowdApp resources) and automatically provisions the supporting infrastructure needed to run them:
- Kubernetes Deployments, Services, and configuration
- Database instances (PostgreSQL)
- Kafka topics and consumers
- Object storage buckets
- In-memory databases (Redis)
- Service mesh and ingress routing
- Dependency resolution between applications
- Application configuration (cdappconfig.json) for service discovery

The environment configuration (ClowdEnvironment) determines which provider implementations to use for each service type, allowing the same application definition to work in local development, testing, and production environments.

## Architecture

### Core Components

- **APIs**: Custom Resource Definitions (CRDs) located in `apis/cloud.redhat.com/v1alpha1/`
  - `ClowdApp`: Main application definition resource
  - `ClowdEnvironment`: Environment configuration resource  
  - `ClowdJobInvocation`: Job execution resource
  - `ClowdAppRef`: Application reference resource

- **Controllers**: Business logic in `controllers/cloud.redhat.com/`
  - `clowdapp_controller.go`: Manages ClowdApp resources
  - `clowdenvironment_controller.go`: Manages ClowdEnvironment resources
  - `clowdjobinvocation_controller.go`: Manages job invocations

- **Providers**: Pluggable service providers in `controllers/cloud.redhat.com/providers/`
  - Database providers (PostgreSQL, shared DB, app interface)
  - Kafka providers (Strimzi, MSK, app interface, managed)
  - Object storage providers (MinIO, app interface)
  - In-memory DB providers (Redis, ElastiCache)
  - Web providers (Caddy gateway, local, default)
  - Metrics, logging, feature flags, and other service providers

### Provider Pattern

Clowder uses a provider-based architecture where different implementations can be swapped based on environment configuration. Each provider implements a common interface and registers itself with the provider registry system.

**Example**: The database provider can be:
- `app-interface`: Use externally managed database (production)
- `local`: Deploy PostgreSQL in the cluster (development/testing)
- `shared`: Use a shared database with schema isolation

The ClowdEnvironment spec determines which provider is active. Applications are written once and work in all environments.

## Development Commands

### Building and Testing
```bash
# Build the manager binary
make build

# Run all tests (requires envtest setup)
make test

# Run specific kuttl test
make kuttl KUTTL_TEST="--test=test-basic-app"

# Format and vet code
make fmt
make vet

# Generate code and manifests
make generate
make manifests

# Generate config types from JSON schema
make genconfig
```

### Local Development
```bash
# Install CRDs into cluster
make install

# Run controller locally (connects to kubeconfig cluster)
make run

# Deploy to cluster
make deploy

# Build and deploy to minikube
make deploy-minikube

# Quick minikube deployment (skips tests)
make deploy-minikube-quick
```

### Container Operations
```bash
# Build container image
make docker-build

# Build without running tests
make docker-build-no-test

# Push to registry
make docker-push

# Push to minikube registry
make docker-push-minikube
```

### Pre-commit Tasks
```bash
# Run all pre-push checks
make pre-push
```

### Environment Setup

Minikube is the recommended local development environment:
```bash
minikube start --cpus 4 --disk-size 36GB --memory 16000MB --driver=kvm2 --addons registry --addons ingress --addons=metrics-server --disable-optimizations
./build/kube_setup.sh
```

### Testing Framework

- Unit tests use Ginkgo/Gomega testing framework
- E2E tests use KUTTL (Kubernetes Test Tool) in `tests/kuttl/`
- Each test scenario has numbered YAML files (00-install.yaml, 01-assert.yaml, etc.)

## Configuration

- **clowder-config.yaml**: Default environment configuration for the operator
- **test_config.json**: Configuration used during `make test` runs
- **controllers/cloud.redhat.com/config/**: App configuration types (generated from JSON schema)
  - `schema.json`: JSON schema defining app config structure
  - `types.go`: Generated Go types (run `make genconfig` to regenerate)
- **ClowdEnvironment resources**: Runtime configuration defining which providers to use

## Key Files and Directories

- `main.go`: Entry point for the controller manager
- `Makefile`: Build system with all development commands
- `go.mod`: Go module definition with dependencies
- `config/`: Kubernetes manifests and Kustomize configurations
  - `config/crd/bases/`: Generated CRD manifests
  - `config/manager/`: Controller deployment configuration
  - `config/rbac/`: Generated RBAC manifests
- `controllers/cloud.redhat.com/`: Controller reconciliation logic
  - `providers/`: Provider implementations (database, kafka, web, etc.)
  - `config/`: App configuration types (generated from JSON schema)
- `apis/cloud.redhat.com/v1alpha1/`: CRD type definitions
- `schema/`: JSON schema for configuration generation
- `tests/kuttl/`: End-to-end KUTTL tests
- `build/`: Build scripts and utilities
- `docs/`: Documentation including developer guide and API reference
- `hack/`: Development utilities and boilerplate

## Important Notes

- Uses Go 1.24.6 (see `go.mod`)
- Built on Kubernetes controller-runtime v0.19.1
- Uses Kubernetes API v0.34.1
- Supports both podman (preferred) and docker for container operations
- CRD generation is handled by controller-gen
- Webhook support can be disabled via configuration
- Metrics and observability built-in with Prometheus integration
- ENVTEST uses Kubernetes 1.30 for local testing

## Testing Patterns

### Unit Tests
- Located alongside source files (e.g., `controllers/cloud.redhat.com/clowdapp_controller_test.go`)
- Use Ginkgo/Gomega BDD-style testing framework
- Controller tests typically use `envtest` which runs a real API server
- Common pattern: Create resources → Wait for reconciliation → Assert expected state
- Test utilities in `controllers/cloud.redhat.com/providers/conftest/` for common setup

### Running Tests
```bash
# Run all unit tests
make test

# Run tests for a specific package
go test ./controllers/cloud.redhat.com/providers/web/...

# Run with verbose output
go test -v ./controllers/...

# IMPORTANT: Testing code changes with KUTTL
# You MUST deploy your latest changes to minikube before running KUTTL tests
# Otherwise you'll be testing the old code that's already deployed!
make deploy-minikube-quick  # Deploy latest changes to minikube
make kuttl                  # Run all kuttl e2e tests

# Run specific kuttl test (after deploying changes)
make deploy-minikube-quick
make kuttl KUTTL_TEST="--test=test-basic-app"
```

### KUTTL E2E Tests
- Located in `tests/kuttl/tests/`
- Each test is a directory with numbered YAML files
- Execution order: `00-*.yaml`, `01-*.yaml`, etc.
- Files ending in `-assert.yaml` contain expected state assertions
- Tests create real ClowdEnvironment and ClowdApp resources
- Default timeout is 30 seconds per step (can be increased in test config)
- Tests run against envtest (simulated API server), not a real cluster

## Provider System Deep Dive

### Provider Lifecycle
1. **Registration**: Providers register themselves in `init()` functions (e.g., `providers/web/default.go`)
2. **Setup**: Called once per reconciliation via `SetupProvider()` to initialize provider-specific resources
3. **Configuration**: Provider populates its section of the app config via provider-specific methods
4. **Reconciliation**: Controllers call provider methods to create/update Kubernetes resources

### Provider Interface Locations
- Base provider interface: `controllers/cloud.redhat.com/providers/provider.go`
- Web provider interface: `controllers/cloud.redhat.com/providers/web/provider.go`
- Database provider interface: `controllers/cloud.redhat.com/providers/database/provider.go`
- Each provider type has its own interface defining required methods

### Adding a New Provider
1. Create provider implementation in appropriate `providers/` subdirectory
2. Implement the provider interface (at minimum: `Provide()` method)
3. Register provider in `init()` function using `providers.ProvidersRegistration.Register()`
4. Add configuration options to JSON schema in `schema/` if needed
5. Run `make genconfig` to regenerate config types
6. Add provider selection logic to environment config

## Dependency Endpoint System

### How Dependencies Work
- ClowdApps can depend on other ClowdApps via `spec.dependencies` field
- Dependencies resolve to connection information (hostname, port, TLS settings)
- Dependency resolution happens in `controllers/cloud.redhat.com/providers/dependencies.go`

### Endpoint Types
- **Public endpoints**: Exposed via `spec.deployments[].web.public.enabled`
- **Private endpoints**: Exposed via `spec.deployments[].web.private.enabled`
- Dependencies can target either public or private endpoints
- Each endpoint can have independent TLS configuration

### Configuration Output
- Resolved dependency info written to app config JSON (cdappconfig.json)
- Mounted into app pods as a config file
- Structure defined in `controllers/cloud.redhat.com/config/` types
- Apps read this config to discover service locations

### Per-Endpoint TLS Configuration
- TLS CA paths are now configured per-endpoint, not globally
- If an environment has TLS ports defined, `tlsCAPath` is populated for each endpoint
- Each dependency endpoint has its own `tlsCAPath` in the app config
- This allows mixing TLS and non-TLS endpoints in the same environment

## Web Provider System

### Provider Types
- **Default**: Standard Kubernetes Service-based routing
- **Local**: Development mode with port-forwarding patterns
- **Operator**: Full-featured with Caddy gateway for ingress

### TLS Handling
- TLS configuration comes from `ClowdEnvironment.spec.providers.web.tls`
- Can be enabled at environment level or per-endpoint
- TLS ports defined in environment (`spec.providers.web.port` for TLS, `spec.providers.web.privatePort` for private)
- Web providers handle TLS termination and certificate mounting

### H2C (HTTP/2 Cleartext) Support
- Enabled via `ClowdApp.spec.deployments[].web.public.h2c: true`
- Allows HTTP/2 over unencrypted connections
- Implemented in web providers by setting pod annotations
- Gateway configurations (like Caddy) use these annotations to enable H2C upstream

### Service Creation Pattern
- Web providers create Services for each deployment's public/private endpoints
- Service names follow pattern: `{app-name}-{deployment-name}[-private]`
- Ports are configured based on environment provider settings
- Service selectors match deployment pod labels

## Common Debugging Patterns

### Reconciliation Issues
1. Check controller logs: `kubectl logs -n clowder-system deployment/clowder-controller-manager`
2. Check resource status: `kubectl describe clowdapp <name> -n <namespace>`
3. Look for events: `kubectl get events -n <namespace> --sort-by='.lastTimestamp'`
4. Verify environment is ready: `kubectl get clowdenvironment <name> -o yaml`

### Configuration Problems
- App config is stored in Secret: `kubectl get secret {appname}-config -o yaml`
- Decode and inspect: `kubectl get secret {appname}-config -o jsonpath='{.data.cdappconfig\.json}' | base64 -d | jq`
- Check provider configuration in ClowdEnvironment spec
- Verify environment deploymentStrategy matches expected mode

### Resource Creation Issues
- Check if provider is running: Look for provider setup logs in controller
- Verify CRDs are installed: `kubectl get crds | grep clowder`
- Check resource ownership: Resources should be owned by ClowdApp/ClowdEnvironment
- Ensure namespace exists and has proper labels

### Test Failures
- Unit test failures: Check if `setup-envtest` is installed and up to date
- KUTTL failures: Check test assertions match actual resource state
- Timing issues: KUTTL has default timeout of 30s, can be increased
- Resource conflicts: Ensure test namespaces are clean between runs

## Code Generation

### When to Regenerate
- After modifying CRD types in `apis/cloud.redhat.com/v1alpha1/`: Run `make generate && make manifests`
- After modifying JSON schema in `schema/`: Run `make genconfig`
- After modifying RBAC markers in controllers: Run `make manifests`

### Generated File Locations
- `zz_generated.deepcopy.go`: Deep copy methods for CRD types
- `config/crd/bases/`: CRD YAML manifests
- `controllers/cloud.redhat.com/config/`: Config types from JSON schema
- `config/rbac/`: RBAC manifests from kubebuilder markers

## Makefile Targets Reference

### Essential Targets
- `make test`: Run all unit tests
- `make build`: Build manager binary
- `make run`: Run controller locally against kubeconfig cluster
- `make deploy`: Deploy to cluster via kustomize
- `make install`: Install CRDs only
- `make uninstall`: Remove CRDs

### Code Quality
- `make fmt`: Format code with gofmt
- `make vet`: Run go vet
- `make lint`: Run golangci-lint
- `make pre-push`: Run all pre-commit checks (fmt, vet, test, build)

### Development Workflow
1. Make code changes
2. Run `make generate` if CRD types changed
3. Run `make manifests` if CRDs or RBAC changed
4. Run `make genconfig` if JSON schema changed
5. Run `make test` to verify tests pass
6. Run `make fmt && make vet` to check code quality
7. Test locally with `make run` or `make deploy-minikube`
8. Run `make pre-push` before committing (runs all checks and regenerates docs)

### Common Pitfalls
- Forgetting to run `make generate` after modifying CRD types → DeepCopy methods won't be updated
- Forgetting to run `make genconfig` after modifying schema.json → Config types will be out of sync
- Modifying generated files directly → Changes will be overwritten on next generation
- Not running `make pre-push` before committing → CI may fail due to missing generated files
- Files that shouldn't be committed: `config/manager/kustomization.yaml`, `controllers/cloud.redhat.com/version.txt` (use `make no-update` to reset)

## Managing Red Hat Konflux Dependency Updates

The `red-hat-konflux` bot regularly opens PRs to update Go dependencies and Docker base images. These PRs typically update `go.mod`, `go.sum`, and occasionally `build/Dockerfile-local`.

### Strategy 1: Merge Individual PRs (Preferred for Small Batches)

When there are many open konflux PRs, use the `gh` CLI to merge them with admin privileges:

```bash
# List all open konflux PRs
gh pr list --author "app/red-hat-konflux" --json number,title,mergeable,state --state open

# Attempt to merge all PRs without conflicts
for pr in 1447 1448 1449 ...; do
  gh pr merge $pr --squash --delete-branch --admin
done
```

**Note**: PRs that are behind the base branch or have conflicts will fail to merge and need conflict resolution.

### Strategy 2: Combine into Single PR (For Large Batches)

When there are 15+ open konflux PRs, combine them into a single PR:

1. **Create a combined branch**:
   ```bash
   git checkout -b combined-konflux-dependency-updates
   ```

2. **Fetch all remote branches**:
   ```bash
   git fetch origin
   ```

3. **Merge all konflux branches sequentially**:
   ```bash
   # Get list of branch names from PRs
   gh pr list --author "app/red-hat-konflux" --json headRefName --state open | jq -r '.[].headRefName'

   # Merge each branch, resolving conflicts as needed
   for branch in <list-of-branches>; do
     git merge --no-edit "origin/$branch" || break
   done
   ```

4. **Resolve merge conflicts**:
   - For dependency conflicts in `go.mod`, choose the newer version
   - For `go.sum`, either manually resolve or use `git checkout --theirs go.sum` and let CI fix it
   - Common conflict pattern: Multiple PRs updating related dependencies (e.g., golang.org/x packages)

5. **Important: Preserve critical dependencies**:
   - **Always keep `rhc-osdk-utils` at the version specified in master** (currently v0.14.0)
   - If a merge downgrades this package, manually revert it back

6. **Push and create PR**:
   ```bash
   git push -u origin combined-konflux-dependency-updates
   gh pr create --title "chore(deps): Combined dependency updates from red-hat-konflux" --body "<detailed summary>"
   ```

7. **Close individual PRs**:
   ```bash
   for pr in <list-of-pr-numbers>; do
     gh pr close $pr --comment "Closing in favor of consolidated PR #<combined-pr-number>"
   done
   ```

### Handling Merge Conflicts in Konflux PRs

When PRs have conflicts (typically after other PRs have been merged):

1. **Checkout the PR branch**:
   ```bash
   gh pr checkout <pr-number>
   ```

2. **Merge master into the PR branch**:
   ```bash
   git merge origin/master
   ```

3. **Resolve conflicts**:
   - **For `go.mod`**: Choose the newer version of each dependency
   - **For `go.sum`**: Use `git checkout --theirs go.sum` to take master's version, then let the merge commit fix it
   - **Pattern**: When choosing between versions, use the higher version number or more recent timestamp

4. **Commit and push**:
   ```bash
   git add go.mod go.sum
   git commit -m "Merge master into PR #<pr-number> and resolve conflicts"
   git push origin <branch-name>
   ```

5. **Wait a few seconds, then merge**:
   ```bash
   sleep 3
   gh pr merge <pr-number> --squash --delete-branch --admin
   ```

### Workflow for Batch Merging Conflicted PRs

When multiple PRs have conflicts, resolve them **in order from oldest to newest**:

```bash
# Work through PRs sequentially (oldest first)
for pr in 1455 1456 1457 1459 1462 1463 1464; do
  echo "Processing PR #$pr"

  # Update local master
  git checkout master && git pull origin master

  # Checkout PR and merge master
  gh pr checkout $pr
  git merge origin/master

  # Resolve conflicts (manual step)
  # Edit go.mod to choose newer versions
  git checkout --theirs go.sum  # Use master's go.sum

  # Commit and push
  git add go.mod go.sum
  git commit -m "Merge master into PR #$pr and resolve conflicts"
  git push origin <branch-name>

  # Merge the PR
  sleep 3
  gh pr merge $pr --squash --delete-branch --admin
done
```

**Why oldest first?**: Earlier PRs may update dependencies that later PRs also touch. Merging in order minimizes cascading conflicts.

### Important Considerations

- **rhc-osdk-utils**: Never allow this to be downgraded - always verify it stays at v0.14.0 (or latest specified version)
- **Go module tidying**: Don't run `go mod tidy` locally if you lack the correct Go version - let CI handle it
- **Conflict resolution strategy**: When in doubt, choose the newer version of dependencies
- **Testing**: Konflux PRs are dependency updates and typically don't require local testing
- **Admin flag**: The `--admin` flag bypasses branch protection rules - use only for automated dependency updates
- **Timing**: Some PRs need a few seconds after pushing before GitHub recognizes them as mergeable