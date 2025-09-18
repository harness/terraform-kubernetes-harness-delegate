package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelegateWithProxyConfiguration(t *testing.T) {
	t.Parallel()

	// Generate unique resource names for parallel testing
	uniqueID := random.UniqueId()
	namespaceName := fmt.Sprintf("test-proxy-delegate-%s", strings.ToLower(uniqueID))
	delegateName := fmt.Sprintf("test-proxy-delegate-%s", strings.ToLower(uniqueID))

	// Setup the terraform options with proxy configuration
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"namespace":        namespaceName,
			"delegate_name":    delegateName,
			"account_id":       "test_account_id",
			"delegate_token":   "test_token",
			"manager_endpoint": "https://app.harness.io",
			"replicas":         1,
			"upgrader_enabled": false,
			"create_namespace": true,
			// Proxy configuration
			"proxy_host":     "proxy.company.com",
			"proxy_port":     "8080",
			"proxy_scheme":   "http",
			"proxy_user":     "proxy_user",
			"proxy_password": "proxy_password",
			"no_proxy":       ".company.com,localhost",
		},
	})

	// Clean up resources with "defer" so that they run even if the test fails
	defer terraform.Destroy(t, terraformOptions)

	// Run terraform init and apply
	terraform.InitAndApply(t, terraformOptions)

	// Get the Kubernetes config path
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)

	// Wait for the namespace to be created
	k8s.WaitUntilNamespaceAvailable(t, kubectlOptions, namespaceName, 10, 3*time.Second)

	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 10, 30*time.Second)

	// Verify the deployment exists and has proxy configuration
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	assert.Equal(t, delegateName, deployment.Name)

	// Verify proxy environment variables in the deployment
	containers := deployment.Spec.Template.Spec.Containers
	require.Greater(t, len(containers), 0, "Deployment should have at least one container")

	// Check for proxy environment variables
	envVars := containers[0].Env
	envMap := make(map[string]string)
	for _, env := range envVars {
		if env.Value != "" {
			envMap[env.Name] = env.Value
		}
	}

	// Verify proxy configuration environment variables
	assert.Equal(t, "proxy.company.com", envMap["PROXY_HOST"])
	assert.Equal(t, "8080", envMap["PROXY_PORT"])
	assert.Equal(t, "http", envMap["PROXY_SCHEME"])
	assert.Equal(t, "proxy_user", envMap["PROXY_USER"])
	assert.Equal(t, "proxy_password", envMap["PROXY_PASSWORD"])
	assert.Equal(t, ".company.com,localhost", envMap["NO_PROXY"])

	// Verify the deployment is running successfully with proxy config
	assert.Equal(t, int32(1), deployment.Status.ReadyReplicas)

	// Verify terraform output contains proxy configuration
	output := terraform.Output(t, terraformOptions, "values")
	assert.NotEmpty(t, output, "Terraform output should not be empty")
	assert.Contains(t, output, "proxy.company.com", "Output should contain proxy host")
	assert.Contains(t, output, "8080", "Output should contain proxy port")
}

func TestDelegateWithoutProxyConfiguration(t *testing.T) {
	t.Parallel()

	// Generate unique resource names for parallel testing
	uniqueID := random.UniqueId()
	namespaceName := fmt.Sprintf("test-no-proxy-delegate-%s", strings.ToLower(uniqueID))
	delegateName := fmt.Sprintf("test-no-proxy-delegate-%s", strings.ToLower(uniqueID))

	// Setup the terraform options without proxy configuration
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"namespace":        namespaceName,
			"delegate_name":    delegateName,
			"account_id":       "test_account_id",
			"delegate_token":   "test_token",
			"manager_endpoint": "https://app.harness.io",
			"replicas":         1,
			"upgrader_enabled": false,
			"create_namespace": true,
			// Explicitly empty proxy configuration
			"proxy_host":     "",
			"proxy_port":     "",
			"proxy_scheme":   "",
			"proxy_user":     "",
			"proxy_password": "",
			"no_proxy":       "",
		},
	})

	// Clean up resources with "defer" so that they run even if the test fails
	defer terraform.Destroy(t, terraformOptions)

	// Run terraform init and apply
	terraform.InitAndApply(t, terraformOptions)

	// Get the Kubernetes config path
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)

	// Wait for the namespace to be created
	k8s.WaitUntilNamespaceAvailable(t, kubectlOptions, namespaceName, 10, 3*time.Second)

	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 10, 30*time.Second)

	// Verify the deployment exists
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	assert.Equal(t, delegateName, deployment.Name)

	// Verify NO proxy environment variables in the deployment
	containers := deployment.Spec.Template.Spec.Containers
	require.Greater(t, len(containers), 0, "Deployment should have at least one container")

	// Check that proxy environment variables are not set or are empty
	envVars := containers[0].Env
	envMap := make(map[string]string)
	for _, env := range envVars {
		envMap[env.Name] = env.Value
	}

	// Verify proxy configuration environment variables are not present or empty
	proxyEnvVars := []string{"PROXY_HOST", "PROXY_PORT", "PROXY_SCHEME", "PROXY_USER", "PROXY_PASSWORD", "NO_PROXY"}
	for _, envVar := range proxyEnvVars {
		if val, exists := envMap[envVar]; exists {
			assert.Empty(t, val, fmt.Sprintf("Environment variable %s should be empty when proxy is not configured", envVar))
		}
	}

	// Verify the deployment is running successfully without proxy config
	assert.Equal(t, int32(1), deployment.Status.ReadyReplicas)
}