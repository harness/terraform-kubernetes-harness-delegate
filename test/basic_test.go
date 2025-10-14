package test

import (
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

func TestBasicDelegateDeployment(t *testing.T) {
	t.Parallel()

	// Load environment variables from .env file
	_ = godotenv.Load(".env")

	// Get unique resource names for parallel testing
	uniqueID := random.UniqueId()
	delegateName := fmt.Sprintf("test-delegate-%s", strings.ToLower(uniqueID))
	namespaceName := os.Getenv("NAMESPACE")
	account_id := os.Getenv("ACCOUNT_ID")
	delegate_token := os.Getenv("DELEGATE_TOKEN")
	delegate_image := os.Getenv("DELEGATE_IMAGE")
	manager_endpoint := os.Getenv("MANAGER_ENDPOINT")
	replicas := 1

	// Setup the terraform options
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
			"upgrader_enabled": false,
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

	// Verify the Helm release exists
	ValidateHelmRelease(t, kubectlOptions, namespaceName, delegateName)

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

	// Verify the service account exists
	serviceAccountName := delegateName
	serviceAccount := k8s.GetServiceAccount(t, kubectlOptions, serviceAccountName)
	assert.Equal(t, serviceAccountName, serviceAccount.Name)

	// Verify terraform output
	output := terraform.Output(t, terraformOptions, "values")
	assert.NotEmpty(t, output, "Terraform output should not be empty")

	// Verify terraform output contains delegate configuration
	assert.Contains(t, output, "delegate_name", "Output should contain delegate_name")
	assert.Contains(t, output, "account_id", "Output should contain account_id")
	assert.Contains(t, output, "delegate_token", "Output should contain delegate_token")
	assert.Contains(t, output, "delegate_image", "Output should contain delegate_image")
	assert.Contains(t, output, "manager_endpoint", "Output should contain manager_endpoint")
}
