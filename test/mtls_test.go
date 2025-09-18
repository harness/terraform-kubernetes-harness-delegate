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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDelegateWithMTLSConfiguration(t *testing.T) {
	t.Parallel()

	// Generate unique resource names for parallel testing
	uniqueID := random.UniqueId()
	namespaceName := fmt.Sprintf("test-mtls-delegate-%s", strings.ToLower(uniqueID))
	delegateName := fmt.Sprintf("test-mtls-delegate-%s", strings.ToLower(uniqueID))
	mtlsSecretName := fmt.Sprintf("test-mtls-secret-%s", strings.ToLower(uniqueID))

	// Get the Kubernetes config path for pre-setup
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)

	// Create the namespace first
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	k8s.CreateNamespace(t, kubectlOptions, namespace)

	// Create a test mTLS secret before running terraform
	mtlsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mtlsSecretName,
			Namespace: namespaceName,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-cert-data\n-----END CERTIFICATE-----"),
			"tls.key": []byte("-----BEGIN PRIVATE KEY-----\ntest-key-data\n-----END PRIVATE KEY-----"),
			"ca.crt":  []byte("-----BEGIN CERTIFICATE-----\ntest-ca-data\n-----END CERTIFICATE-----"),
		},
	}
	k8s.CreateSecret(t, kubectlOptions, mtlsSecret)

	// Setup the terraform options with mTLS configuration
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"namespace":         namespaceName,
			"delegate_name":     delegateName,
			"account_id":        "test_account_id",
			"delegate_token":    "test_token",
			"manager_endpoint":  "https://app.harness.io",
			"replicas":          1,
			"upgrader_enabled":  false,
			"create_namespace":  false, // We created it manually
			"mtls_secret_name":  mtlsSecretName,
		},
	})

	// Clean up resources with "defer" so that they run even if the test fails
	defer func() {
		terraform.Destroy(t, terraformOptions)
		// Clean up the manually created resources
		k8s.DeleteSecret(t, kubectlOptions, mtlsSecretName)
		k8s.DeleteNamespace(t, kubectlOptions, namespaceName)
	}()

	// Run terraform init and apply
	terraform.InitAndApply(t, terraformOptions)

	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 10, 30*time.Second)

	// Verify the deployment exists
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	assert.Equal(t, delegateName, deployment.Name)

	// Verify mTLS secret is mounted as volume
	containers := deployment.Spec.Template.Spec.Containers
	require.Greater(t, len(containers), 0, "Deployment should have at least one container")

	// Check for volume mounts related to mTLS
	volumeMounts := containers[0].VolumeMounts
	var mtlsVolumeMount *corev1.VolumeMount
	for _, vm := range volumeMounts {
		if strings.Contains(vm.Name, "mtls") || strings.Contains(vm.Name, "tls") {
			mtlsVolumeMount = &vm
			break
		}
	}

	if mtlsVolumeMount != nil {
		assert.NotEmpty(t, mtlsVolumeMount.MountPath, "mTLS volume should have a mount path")
		assert.True(t, mtlsVolumeMount.ReadOnly, "mTLS volume should be read-only")
	}

	// Check for volumes in the pod template
	volumes := deployment.Spec.Template.Spec.Volumes
	var mtlsVolume *corev1.Volume
	for _, vol := range volumes {
		if vol.Secret != nil && vol.Secret.SecretName == mtlsSecretName {
			mtlsVolume = &vol
			break
		}
	}

	if mtlsVolume != nil {
		assert.Equal(t, mtlsSecretName, mtlsVolume.Secret.SecretName, "Volume should reference the correct mTLS secret")
	}

	// Verify the deployment is running successfully with mTLS config
	assert.Equal(t, int32(1), deployment.Status.ReadyReplicas)

	// Verify terraform output contains mTLS configuration
	output := terraform.Output(t, terraformOptions, "values")
	assert.NotEmpty(t, output, "Terraform output should not be empty")
	assert.Contains(t, output, mtlsSecretName, "Output should contain mTLS secret name")

	// Verify the mTLS secret still exists and is accessible
	secret := k8s.GetSecret(t, kubectlOptions, mtlsSecretName)
	assert.Equal(t, mtlsSecretName, secret.Name)
	assert.Equal(t, corev1.SecretTypeTLS, secret.Type)
	assert.NotEmpty(t, secret.Data["tls.crt"], "Secret should contain certificate")
	assert.NotEmpty(t, secret.Data["tls.key"], "Secret should contain private key")
}

func TestDelegateWithoutMTLSConfiguration(t *testing.T) {
	t.Parallel()

	// Generate unique resource names for parallel testing
	uniqueID := random.UniqueId()
	namespaceName := fmt.Sprintf("test-no-mtls-delegate-%s", strings.ToLower(uniqueID))
	delegateName := fmt.Sprintf("test-no-mtls-delegate-%s", strings.ToLower(uniqueID))

	// Setup the terraform options without mTLS configuration
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
			"mtls_secret_name": "", // Empty mTLS secret name
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

	// Verify NO mTLS volumes are mounted
	containers := deployment.Spec.Template.Spec.Containers
	require.Greater(t, len(containers), 0, "Deployment should have at least one container")

	// Check that no mTLS-related volume mounts exist
	volumeMounts := containers[0].VolumeMounts
	for _, vm := range volumeMounts {
		assert.False(t, strings.Contains(strings.ToLower(vm.Name), "mtls"), 
			fmt.Sprintf("Should not have mTLS volume mount: %s", vm.Name))
		assert.False(t, strings.Contains(strings.ToLower(vm.Name), "tls"), 
			fmt.Sprintf("Should not have TLS volume mount: %s", vm.Name))
	}

	// Check that no mTLS-related volumes exist
	volumes := deployment.Spec.Template.Spec.Volumes
	for _, vol := range volumes {
		if vol.Secret != nil {
			assert.False(t, strings.Contains(strings.ToLower(vol.Secret.SecretName), "mtls"), 
				fmt.Sprintf("Should not have mTLS secret volume: %s", vol.Secret.SecretName))
		}
	}

	// Verify the deployment is running successfully without mTLS config
	assert.Equal(t, int32(1), deployment.Status.ReadyReplicas)

	// Verify terraform output does not contain mTLS configuration
	output := terraform.Output(t, terraformOptions, "values")
	assert.NotEmpty(t, output, "Terraform output should not be empty")
	// The output should not contain any mTLS secret references
	assert.NotContains(t, strings.ToLower(output), "mtls", "Output should not contain mTLS references")
}