package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultTerraformVars returns a map of default terraform variables for testing
func DefaultTerraformVars(namespaceName, delegateName string) map[string]interface{} {
	return map[string]interface{}{
		"namespace":        namespaceName,
		"delegate_name":    delegateName,
		"account_id":       "test_account_id",
		"delegate_token":   "test_token",
		"manager_endpoint": "https://app.harness.io",
		"replicas":         1,
		"upgrader_enabled": false,
		"create_namespace": true,
	}
}

// CreateTerraformOptions creates terraform options with default retry settings
func CreateTerraformOptions(t *testing.T, vars map[string]interface{}) *terraform.Options {
	return terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars:         vars,
	})
}

// CreateTestSecret creates a test Kubernetes secret for testing purposes
func CreateTestSecret(t *testing.T, kubectlOptions *k8s.KubectlOptions, secretName, secretType string, data map[string][]byte) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: kubectlOptions.Namespace,
		},
		Type: corev1.SecretType(secretType),
		Data: data,
	}
	k8s.CreateSecret(t, kubectlOptions, secret)
}

// CreateMTLSTestSecret creates a test mTLS secret with dummy certificates
func CreateMTLSTestSecret(t *testing.T, kubectlOptions *k8s.KubectlOptions, secretName string) {
	data := map[string][]byte{
		"tls.crt": []byte("-----BEGIN CERTIFICATE-----\ntest-cert-data\n-----END CERTIFICATE-----"),
		"tls.key": []byte("-----BEGIN PRIVATE KEY-----\ntest-key-data\n-----END PRIVATE KEY-----"),
		"ca.crt":  []byte("-----BEGIN CERTIFICATE-----\ntest-ca-data\n-----END CERTIFICATE-----"),
	}
	CreateTestSecret(t, kubectlOptions, secretName, string(corev1.SecretTypeTLS), data)
}

// WaitForDelegateDeployment waits for the delegate deployment to be ready and validates basic requirements
func WaitForDelegateDeployment(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, timeout time.Duration) {
	// Wait for the namespace to be available
	k8s.WaitUntilNamespaceAvailable(t, kubectlOptions, kubectlOptions.Namespace, 10, 3*time.Second)
	
	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 20, timeout)
	
	// Verify the deployment exists
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	require.Equal(t, delegateName, deployment.Name)
	require.Greater(t, len(deployment.Spec.Template.Spec.Containers), 0, "Deployment should have at least one container")
}

// GetEnvironmentVariables extracts environment variables from a deployment into a map
func GetEnvironmentVariables(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string) map[string]string {
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	require.Greater(t, len(deployment.Spec.Template.Spec.Containers), 0, "Deployment should have at least one container")
	
	envVars := deployment.Spec.Template.Spec.Containers[0].Env
	envMap := make(map[string]string)
	for _, env := range envVars {
		if env.Value != "" {
			envMap[env.Name] = env.Value
		}
	}
	return envMap
}

// ValidateBasicDelegateConfiguration validates that basic delegate configuration is present
func ValidateBasicDelegateConfiguration(t *testing.T, envMap map[string]string, expectedAccountID, expectedManagerEndpoint, expectedDelegateName string) {
	require.Equal(t, expectedAccountID, envMap["ACCOUNT_ID"], "Account ID should match")
	require.Equal(t, expectedManagerEndpoint, envMap["MANAGER_HOST_AND_PORT"], "Manager endpoint should match")
	require.Equal(t, expectedDelegateName, envMap["DELEGATE_NAME"], "Delegate name should match")
}

// ValidateProxyConfiguration validates that proxy environment variables are correctly set
func ValidateProxyConfiguration(t *testing.T, envMap map[string]string, expectedProxy ProxyConfig) {
	if expectedProxy.Host != "" {
		require.Equal(t, expectedProxy.Host, envMap["PROXY_HOST"], "Proxy host should match")
		require.Equal(t, expectedProxy.Port, envMap["PROXY_PORT"], "Proxy port should match")
		require.Equal(t, expectedProxy.Scheme, envMap["PROXY_SCHEME"], "Proxy scheme should match")
		
		if expectedProxy.User != "" {
			require.Equal(t, expectedProxy.User, envMap["PROXY_USER"], "Proxy user should match")
		}
		if expectedProxy.Password != "" {
			require.Equal(t, expectedProxy.Password, envMap["PROXY_PASSWORD"], "Proxy password should match")
		}
		if expectedProxy.NoProxy != "" {
			require.Equal(t, expectedProxy.NoProxy, envMap["NO_PROXY"], "No proxy should match")
		}
	}
}

// ValidateNoProxyConfiguration validates that no proxy environment variables are set
func ValidateNoProxyConfiguration(t *testing.T, envMap map[string]string) {
	proxyEnvVars := []string{"PROXY_HOST", "PROXY_PORT", "PROXY_SCHEME", "PROXY_USER", "PROXY_PASSWORD", "NO_PROXY"}
	for _, envVar := range proxyEnvVars {
		if val, exists := envMap[envVar]; exists {
			require.Empty(t, val, fmt.Sprintf("Environment variable %s should be empty when proxy is not configured", envVar))
		}
	}
}

// ValidateMTLSVolumeConfiguration validates that mTLS volumes are properly configured
func ValidateMTLSVolumeConfiguration(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName, expectedSecretName string) {
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	
	// Check for volumes in the pod template
	volumes := deployment.Spec.Template.Spec.Volumes
	var mtlsVolume *corev1.Volume
	for _, vol := range volumes {
		if vol.Secret != nil && vol.Secret.SecretName == expectedSecretName {
			mtlsVolume = &vol
			break
		}
	}
	
	require.NotNil(t, mtlsVolume, "mTLS volume should exist")
	require.Equal(t, expectedSecretName, mtlsVolume.Secret.SecretName, "Volume should reference the correct mTLS secret")
	
	// Check for volume mounts
	containers := deployment.Spec.Template.Spec.Containers
	require.Greater(t, len(containers), 0, "Deployment should have at least one container")
	
	volumeMounts := containers[0].VolumeMounts
	var mtlsVolumeMount *corev1.VolumeMount
	for _, vm := range volumeMounts {
		if strings.Contains(vm.Name, "mtls") || strings.Contains(vm.Name, "tls") {
			mtlsVolumeMount = &vm
			break
		}
	}
	
	if mtlsVolumeMount != nil {
		require.NotEmpty(t, mtlsVolumeMount.MountPath, "mTLS volume should have a mount path")
		require.True(t, mtlsVolumeMount.ReadOnly, "mTLS volume should be read-only")
	}
}

// ValidateNoMTLSConfiguration validates that no mTLS volumes are configured
func ValidateNoMTLSConfiguration(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string) {
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	
	containers := deployment.Spec.Template.Spec.Containers
	require.Greater(t, len(containers), 0, "Deployment should have at least one container")
	
	// Check that no mTLS-related volume mounts exist
	volumeMounts := containers[0].VolumeMounts
	for _, vm := range volumeMounts {
		require.False(t, strings.Contains(strings.ToLower(vm.Name), "mtls"), 
			fmt.Sprintf("Should not have mTLS volume mount: %s", vm.Name))
		require.False(t, strings.Contains(strings.ToLower(vm.Name), "tls"), 
			fmt.Sprintf("Should not have TLS volume mount: %s", vm.Name))
	}
	
	// Check that no mTLS-related volumes exist
	volumes := deployment.Spec.Template.Spec.Volumes
	for _, vol := range volumes {
		if vol.Secret != nil {
			require.False(t, strings.Contains(strings.ToLower(vol.Secret.SecretName), "mtls"), 
				fmt.Sprintf("Should not have mTLS secret volume: %s", vol.Secret.SecretName))
		}
	}
}

// ProxyConfig represents proxy configuration for testing
type ProxyConfig struct {
	Host     string
	Port     string
	Scheme   string
	User     string
	Password string
	NoProxy  string
}

// CleanupResources performs cleanup of test resources
func CleanupResources(t *testing.T, terraformOptions *terraform.Options, kubectlOptions *k8s.KubectlOptions, additionalSecrets []string) {
	// Destroy terraform resources
	terraform.Destroy(t, terraformOptions)
	
	// Clean up any additional secrets created manually
	for _, secretName := range additionalSecrets {
		if secretName != "" {
			k8s.DeleteSecret(t, kubectlOptions, secretName)
		}
	}
	
	// Clean up namespace if it was created manually
	if kubectlOptions.Namespace != "" {
		k8s.DeleteNamespace(t, kubectlOptions, kubectlOptions.Namespace)
	}
}

// ValidateHelmRelease validates that the Helm release is properly deployed
func ValidateHelmRelease(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string) {
	// This function can be expanded to validate Helm release status
	// For now, we'll validate that the deployment exists and is ready
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	require.Equal(t, delegateName, deployment.Name)
	require.Equal(t, int32(1), deployment.Status.ReadyReplicas, "Deployment should have ready replicas")
}

// GenerateUniqueTestNames generates unique test resource names
func GenerateUniqueTestNames(prefix, uniqueID string) (namespaceName, delegateName string) {
	namespaceName = fmt.Sprintf("%s-%s", prefix, strings.ToLower(uniqueID))
	delegateName = fmt.Sprintf("%s-delegate-%s", prefix, strings.ToLower(uniqueID))
	return namespaceName, delegateName
}