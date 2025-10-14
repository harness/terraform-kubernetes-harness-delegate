package test

import (
	// "encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDelegateWithUpgrader(t *testing.T) {
	t.Parallel()

	// Load environment variables from .env file
	_ = godotenv.Load(".env")

	// Get unique resource names for parallel testing
	uniqueID := random.UniqueId()
	delegateName := fmt.Sprintf("test-delegate-%s", strings.ToLower(uniqueID))
	namespaceName := "harness-delegate-ng"
	account_id := os.Getenv("ACCOUNT_ID")
	delegate_token := os.Getenv("DELEGATE_TOKEN")
	delegate_image := os.Getenv("DELEGATE_IMAGE")
	manager_endpoint := os.Getenv("MANAGER_ENDPOINT")
	replicas := 2

	// Setup the terraform options with proxy configuration
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"namespace":        namespaceName,
			"delegate_name":    delegateName,
			"account_id":       account_id,
			"delegate_token":   delegate_token,
			"delegate_image":   delegate_image,
			"manager_endpoint": manager_endpoint,
			"replicas":         replicas,
			"upgrader_enabled": true,
			"create_namespace": true,
		},
	})

	// Clean up resources if the test fails
	t.Cleanup(func() {
		terraform.Destroy(t, terraformOptions)
	})

	// Run terraform init and apply
	terraform.InitAndApply(t, terraformOptions)

	// Get the Kubernetes config path
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)

	// Verify the namespace exists
	namespace := k8s.GetNamespace(t, kubectlOptions, namespaceName)
	assert.Equal(t, namespaceName, namespace.Name)

	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 10, 30*time.Second)

	// Verify the deployment exists and has the correct replicas
	deploymentName := delegateName
	deployment := k8s.GetDeployment(t, kubectlOptions, deploymentName)
	assert.Equal(t, deploymentName, deployment.Name)
	assert.Equal(t, (int32)(replicas), *deployment.Spec.Replicas)
	assert.Equal(t, (int32)(replicas), deployment.Status.ReadyReplicas)

	// Getting pod list
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
	pods := k8s.ListPods(t, kubectlOptions, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	assert.Equal(t, replicas, len(pods), "expected number of pods")

	// Verify container with correct configuration
	containers := pods[0].Spec.Containers
	require.Greater(t, len(containers), 0, "Pod should have at least one container")

	container := containers[0]
	envMap := ResolveContainerEnvMap(t, kubectlOptions, container)
	
	// Validate basic delegate configuration
	ValidateBasicDelegateConfiguration(t, envMap, account_id, manager_endpoint, delegateName, &container, delegate_image)

	// Validate upgrader resources
	ValidateUpgraderResources(t, kubectlOptions, delegateName)

	output := terraform.Output(t, terraformOptions, "values")
	assert.NotEmpty(t, output, "Terraform output should not be empty")
	
	// Verify terraform output contains upgrader configuration
	assert.Contains(t, output, "upgrader_enabled", "Output should contain upgrader_enabled")
}

func TestDelegateWithUpgraderProxy(t *testing.T) {
	t.Parallel()

	// Load environment variables from .env file
	_ = godotenv.Load(".env")

	// Get unique resource names for parallel testing
	uniqueID := random.UniqueId()
	delegateName := fmt.Sprintf("test-delegate-%s", strings.ToLower(uniqueID))
	namespaceName := "harness-delegate-ng"
	account_id := os.Getenv("ACCOUNT_ID")
	delegate_token := os.Getenv("DELEGATE_TOKEN")
	delegate_image := os.Getenv("DELEGATE_IMAGE")
	manager_endpoint := os.Getenv("MANAGER_ENDPOINT")
	replicas := 2
	proxy_host := os.Getenv("PROXY_HOST")
	proxy_port := os.Getenv("PROXY_PORT")
	proxy_scheme := os.Getenv("PROXY_SCHEME")
	proxy_user := os.Getenv("PROXY_USER")
	proxy_password := os.Getenv("PROXY_PASSWORD")
	no_proxy := os.Getenv("NO_PROXY")

	// Setup the terraform options with proxy configuration
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"namespace":        namespaceName,
			"delegate_name":    delegateName,
			"account_id":       account_id,
			"delegate_token":   delegate_token,
			"delegate_image":   delegate_image,
			"manager_endpoint": manager_endpoint,
			"replicas":         replicas,
			"upgrader_enabled": true,
			"create_namespace": true,
			// Proxy configuration
			"proxy_host":     proxy_host,
			"proxy_port":     proxy_port,
			"proxy_scheme":   proxy_scheme,
			"proxy_user":     proxy_user,
			"proxy_password": proxy_password,
			"no_proxy":       no_proxy,
		},
	})

	// Clean up resources if the test fails
	t.Cleanup(func() {
		terraform.Destroy(t, terraformOptions)
	})

	// Run terraform init and apply
	terraform.InitAndApply(t, terraformOptions)

	// Get the Kubernetes config path
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)

	// Verify the namespace exists
	namespace := k8s.GetNamespace(t, kubectlOptions, namespaceName)
	assert.Equal(t, namespaceName, namespace.Name)

	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 10, 30*time.Second)

	// Verify the deployment exists and has the correct replicas
	deploymentName := delegateName
	deployment := k8s.GetDeployment(t, kubectlOptions, deploymentName)
	assert.Equal(t, deploymentName, deployment.Name)
	assert.Equal(t, (int32)(replicas), *deployment.Spec.Replicas)
	assert.Equal(t, (int32)(replicas), deployment.Status.ReadyReplicas)

	// Getting pod list
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
	pods := k8s.ListPods(t, kubectlOptions, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	assert.Equal(t, replicas, len(pods), "expected number of pods")

	// Verify container with correct configuration
	containers := pods[0].Spec.Containers
	require.Greater(t, len(containers), 0, "Pod should have at least one container")

	container := containers[0]
	envMap := ResolveContainerEnvMap(t, kubectlOptions, container)
	
	// Validate basic delegate configuration
	ValidateBasicDelegateConfiguration(t, envMap, account_id, manager_endpoint, delegateName, &container, delegate_image)

	// Validate proxy configuration
	proxyConfig := ProxyConfig{
		Host:     proxy_host,
		Port:     proxy_port,
		Scheme:   proxy_scheme,
		User:     proxy_user,
		Password: proxy_password,
		NoProxy:  no_proxy,
	}

	ValidateProxyConfiguration(t, envMap, proxyConfig)

	// Verify ConfigMap exists
	configMapName := fmt.Sprintf("%s-proxy", delegateName)
	configMap := k8s.GetConfigMap(t, kubectlOptions, configMapName)
	assert.Equal(t, configMapName, configMap.Name)

	// Verify Secret exists
	secretName := fmt.Sprintf("%s-proxy", delegateName)
	secret := k8s.GetSecret(t, kubectlOptions, secretName)
	assert.Equal(t, secretName, secret.Name)

	// Validate upgrader resources
	ValidateUpgraderResources(t, kubectlOptions, delegateName)
	
	output := terraform.Output(t, terraformOptions, "values")
	assert.NotEmpty(t, output, "Terraform output should not be empty")

	// Verify terraform output contains proxy configuration
	assert.Contains(t, output, "proxy_host", "Output should contain proxy host")
	assert.Contains(t, output, "proxy_port", "Output should contain proxy port")
	assert.Contains(t, output, "proxy_scheme", "Output should contain proxy scheme")
	assert.Contains(t, output, "proxy_user", "Output should contain proxy user")
	assert.Contains(t, output, "proxy_password", "Output should contain proxy password")
	assert.Contains(t, output, "no_proxy", "Output should contain no proxy")

	// Verify terraform output contains upgrader configuration
	assert.Contains(t, output, "upgrader_enabled", "Output should contain upgrader_enabled")
}
