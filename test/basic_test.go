package test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBasicDelegateDeployment(t *testing.T) {
	t.Parallel()

	// Generate unique resource names for parallel testing
	uniqueID := random.UniqueId()
	namespaceName := fmt.Sprintf("test-harness-delegate-%s", strings.ToLower(uniqueID))
	delegateName := fmt.Sprintf("test-delegate-%s", strings.ToLower(uniqueID))

	// Setup the terraform options
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

	// Verify the namespace exists
	namespace := k8s.GetNamespace(t, kubectlOptions, namespaceName)
	assert.Equal(t, namespaceName, namespace.Name)

	// Verify the Helm release exists
	helmOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
	}

	// Check if the Helm release exists and is deployed
	releases := helm.GetReleases(t, helmOptions, true, "")
	var foundRelease bool
	for _, release := range releases {
		if release.Name == delegateName && release.Namespace == namespaceName {
			foundRelease = true
			assert.Equal(t, "deployed", release.Status)
			break
		}
	}
	require.True(t, foundRelease, "Helm release should exist and be deployed")

	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 10, 30*time.Second)

	// Verify the deployment exists and has the correct replicas
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	assert.Equal(t, delegateName, deployment.Name)
	assert.Equal(t, int32(1), *deployment.Spec.Replicas)
	assert.Equal(t, int32(1), deployment.Status.ReadyReplicas)

	// Verify ConfigMap exists with correct configuration
	configMapName := fmt.Sprintf("%s-config", delegateName)
	configMap := k8s.GetConfigMap(t, kubectlOptions, configMapName)
	assert.Equal(t, configMapName, configMap.Name)

	// Verify environment variables in the deployment
	containers := deployment.Spec.Template.Spec.Containers
	require.Greater(t, len(containers), 0, "Deployment should have at least one container")

	// Check for essential environment variables
	envVars := containers[0].Env
	envMap := make(map[string]string)
	for _, env := range envVars {
		envMap[env.Name] = env.Value
	}

	assert.Equal(t, "test_account_id", envMap["ACCOUNT_ID"])
	assert.Equal(t, "https://app.harness.io", envMap["MANAGER_HOST_AND_PORT"])
	assert.Equal(t, delegateName, envMap["DELEGATE_NAME"])

	// Verify the service account exists
	serviceAccountName := delegateName
	serviceAccount := k8s.GetServiceAccount(t, kubectlOptions, serviceAccountName)
	assert.Equal(t, serviceAccountName, serviceAccount.Name)

	// Verify terraform output
	output := terraform.Output(t, terraformOptions, "values")
	assert.NotEmpty(t, output, "Terraform output should not be empty")
}