# Terratest Tests for Terraform Kubernetes Harness Delegate

This directory contains comprehensive Terratest unit tests for the Terraform Kubernetes Harness Delegate module.

## Overview

The tests validate various deployment scenarios and configurations of the Harness delegate Helm chart using Terratest, ensuring that the Terraform module works correctly across different use cases.

## Test Files

- **`basic_test.go`** - Tests basic delegate deployment functionality
- **`proxy_test.go`** - Tests proxy configuration scenarios (with and without proxy)
- **`mtls_test.go`** - Tests mTLS configuration scenarios (with and without mTLS)
- **`helpers.go`** - Shared utility functions and helpers for all tests

## Prerequisites

### Software Requirements

1. **Go 1.21+** - Required to run the tests
2. **Terraform** - To provision infrastructure
3. **kubectl** - To interact with Kubernetes cluster
4. **Helm** - For Helm chart operations
5. **Access to a Kubernetes cluster** - Either local (minikube, kind) or cloud-based

### Environment Setup

1. **Kubernetes Cluster Access**
   ```bash
   # Ensure kubectl is configured and can access your cluster
   kubectl cluster-info
   ```

2. **Helm Repository Access**
   ```bash
   # Ensure access to Harness Helm repository
   helm repo add harness https://app.harness.io/storage/harness-download/delegate-helm-chart/
   helm repo update
   ```

3. **Go Dependencies**
   ```bash
   # Install Go dependencies
   go mod tidy
   ```

## Running Tests

### Run All Tests

```bash
# Run all tests with verbose output
go test -v ./test/...

# Run tests in parallel (faster execution)
go test -v -parallel 4 ./test/...
```

### Run Specific Tests

```bash
# Run only basic deployment tests
go test -v ./test/ -run TestBasicDelegateDeployment

# Run only proxy tests
go test -v ./test/ -run TestDelegateWithProxyConfiguration

# Run only mTLS tests
go test -v ./test/ -run TestDelegateWithMTLSConfiguration
```

### Run Tests with Timeout

```bash
# Set longer timeout for slower clusters
go test -v -timeout 30m ./test/...
```

## Test Scenarios

### 1. Basic Deployment Test (`basic_test.go`)

**TestBasicDelegateDeployment**
- Validates basic delegate deployment
- Verifies namespace creation
- Checks Helm release status
- Validates deployment readiness
- Confirms environment variables
- Verifies service account creation
- Tests Terraform output

**What it tests:**
- ✅ Namespace creation
- ✅ Helm chart deployment
- ✅ Deployment scaling and readiness
- ✅ Essential environment variables
- ✅ Service account provisioning
- ✅ Terraform output validation

### 2. Proxy Configuration Tests (`proxy_test.go`)

**TestDelegateWithProxyConfiguration**
- Tests delegate deployment with full proxy configuration
- Validates proxy environment variables
- Ensures deployment works with proxy settings

**TestDelegateWithoutProxyConfiguration**
- Tests delegate deployment without proxy configuration
- Validates absence of proxy environment variables
- Ensures clean deployment without proxy settings

**What it tests:**
- ✅ Proxy host, port, scheme configuration
- ✅ Proxy authentication (user/password)
- ✅ No proxy exclusions
- ✅ Clean deployment without proxy settings

### 3. mTLS Configuration Tests (`mtls_test.go`)

**TestDelegateWithMTLSConfiguration**
- Creates test mTLS secret
- Tests delegate deployment with mTLS configuration
- Validates volume mounts and secret references
- Ensures secure communication setup

**TestDelegateWithoutMTLSConfiguration**
- Tests delegate deployment without mTLS configuration
- Validates absence of mTLS volumes and mounts
- Ensures clean deployment without mTLS settings

**What it tests:**
- ✅ mTLS secret creation and reference
- ✅ Volume mounting for certificates
- ✅ Read-only volume configuration
- ✅ Clean deployment without mTLS settings

## Test Configuration

### Default Test Variables

The tests use these default values (defined in `helpers.go`):

```go
{
    "namespace":        "test-harness-delegate-{uniqueID}",
    "delegate_name":    "test-delegate-{uniqueID}",
    "account_id":       "test_account_id",
    "delegate_token":   "test_token",
    "manager_endpoint": "https://app.harness.io",
    "replicas":         1,
    "upgrader_enabled": false,
    "create_namespace": true,
}
```

### Customizing Test Variables

You can modify test variables by editing the respective test files or using environment variables:

```bash
# Example: Run tests with custom timeout
export TERRATEST_TIMEOUT=45m
go test -v ./test/...
```

## Test Utilities

### Helper Functions (`helpers.go`)

The helper file provides reusable functions:

- **`DefaultTerraformVars()`** - Standard terraform variables
- **`CreateTerraformOptions()`** - Terraform options with retry settings
- **`CreateMTLSTestSecret()`** - Creates test mTLS secrets
- **`WaitForDelegateDeployment()`** - Waits for deployment readiness
- **`GetEnvironmentVariables()`** - Extracts env vars from deployment
- **`ValidateBasicDelegateConfiguration()`** - Validates basic config
- **`ValidateProxyConfiguration()`** - Validates proxy settings
- **`ValidateMTLSVolumeConfiguration()`** - Validates mTLS volumes
- **`CleanupResources()`** - Resource cleanup utilities

## Parallel Test Execution

Tests are designed to run in parallel using unique resource names:

```go
uniqueID := random.UniqueId()
namespaceName := fmt.Sprintf("test-delegate-%s", strings.ToLower(uniqueID))
```

This prevents conflicts when running multiple tests simultaneously.

## Troubleshooting

### Common Issues

1. **Kubernetes Cluster Access**
   ```bash
   # Check cluster connectivity
   kubectl get nodes
   
   # Check current context
   kubectl config current-context
   ```

2. **Helm Repository Access**
   ```bash
   # Update Helm repositories
   helm repo update
   
   # Verify chart availability
   helm search repo harness-delegate-ng
   ```

3. **Resource Cleanup**
   ```bash
   # Manual cleanup if tests fail
   kubectl get namespaces | grep test-
   kubectl delete namespace <test-namespace>
   ```

4. **Go Module Issues**
   ```bash
   # Clean and re-download dependencies
   go clean -modcache
   go mod tidy
   ```

### Test Timeout Issues

If tests timeout, try:

```bash
# Increase timeout
go test -v -timeout 45m ./test/...

# Run single test to isolate issues
go test -v -run TestBasicDelegateDeployment ./test/
```

### Debug Mode

Enable debug output:

```bash
# Verbose Terratest output
export TF_LOG=DEBUG
go test -v ./test/...
```


## Best Practices

1. **Resource Naming** - Always use unique IDs for parallel execution
2. **Cleanup** - Use `defer` statements for resource cleanup
3. **Timeouts** - Set appropriate timeouts for your cluster performance
4. **Validation** - Test both positive and negative scenarios
5. **Isolation** - Each test should be independent and not rely on others

## Contributing

When adding new tests:

1. Follow the existing naming conventions
2. Use helper functions from `helpers.go`
3. Include both positive and negative test cases
4. Add proper cleanup with `defer`
5. Update this README with new test descriptions

## Support

For issues or questions:

1. Check the [Terratest documentation](https://terratest.gruntwork.io/)
2. Review the [Harness Delegate documentation](https://docs.harness.io/)
3. Create an issue in the repository