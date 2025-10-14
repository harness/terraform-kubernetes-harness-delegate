package test

import (
	// "context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	// "gopkg.in/yaml.v2"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// HelmRelease models a release from `helm list -o json`
type HelmRelease struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Chart     string `json:"chart"`
	AppVer    string `json:"app_version"`
}

// CreateTerraformOptions creates terraform options with default retry settings
func CreateTerraformOptions(t *testing.T, vars map[string]interface{}) *terraform.Options {
	return terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars:         vars,
	})
}

// ListHelmReleases fetches helm releases in a namespace and returns them as []HelmRelease
func ListHelmReleases(t *testing.T, options *helm.Options, namespace string) []HelmRelease {
	out, err := helm.RunHelmCommandAndGetOutputE(t, options,
		"list", "-n", namespace, "-o", "json")
	require.NoError(t, err)

	var releases []HelmRelease
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &releases))

	return releases
}

// WaitForDelegateDeployment waits for the delegate deployment to be ready and validates basic requirements
func WaitForDelegateDeployment(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, timeout time.Duration) {
	// Wait for the deployment to be ready
	k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, delegateName, 20, timeout)

	// Verify the deployment exists
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	require.Equal(t, delegateName, deployment.Name)
	require.Greater(t, len(deployment.Spec.Template.Spec.Containers), 0, "Deployment should have at least one container")
}

// ValidateBasicDelegateConfiguration validates that basic delegate configuration is present
func ValidateBasicDelegateConfiguration(t *testing.T, envMap map[string]string, expectedAccountID, expectedManagerEndpoint, expectedDelegateName string, container *corev1.Container, expectedImage string) {
	require.Equal(t, expectedAccountID, envMap["ACCOUNT_ID"], "Account ID should match")
	require.Equal(t, expectedManagerEndpoint, envMap["MANAGER_HOST_AND_PORT"], "Manager endpoint should match")
	require.Equal(t, expectedDelegateName, envMap["DELEGATE_NAME"], "Delegate name should match")
	require.Equal(t, expectedImage, container.Image, "Image should match")
}

// ValidateProxyConfiguration validates that proxy environment variables are correctly set
func ValidateProxyConfiguration(t *testing.T, envMap map[string]string, expectedProxy ProxyConfig) {
		require.Equal(t, expectedProxy.Host, envMap["PROXY_HOST"], "Proxy host should match")
		require.Equal(t, expectedProxy.Port, envMap["PROXY_PORT"], "Proxy port should match")
		require.Equal(t, expectedProxy.Scheme, envMap["PROXY_SCHEME"], "Proxy scheme should match")
		require.Equal(t, expectedProxy.NoProxy, envMap["NO_PROXY"], "No proxy should match")
		
		require.Equal(t, expectedProxy.User, base64.StdEncoding.EncodeToString([]byte(envMap["PROXY_USER"])), "Proxy user should match")
		require.Equal(t, expectedProxy.Password, base64.StdEncoding.EncodeToString([]byte(envMap["PROXY_PASSWORD"])), "Proxy password should match")
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

func ValidateUpgraderResources(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string) {
	// Verify ConfigMap exists
	configMapName := fmt.Sprintf("%s-upgrader-config", delegateName)
	configMap := k8s.GetConfigMap(t, kubectlOptions, configMapName)
	require.Equal(t, configMapName, configMap.Name)

	// Verify Secret exists
	secretName := fmt.Sprintf("%s-upgrader-token", delegateName)
	secret := k8s.GetSecret(t, kubectlOptions, secretName)
	require.Equal(t, secretName, secret.Name)

	// Verify the service account exists
	serviceAccountName := fmt.Sprintf("%s-upgrader-cronjob-sa", delegateName)
	serviceAccount := k8s.GetServiceAccount(t, kubectlOptions, serviceAccountName)
	require.Equal(t, serviceAccountName, serviceAccount.Name)

	// Verify the cronjob exists
	cronjobName := fmt.Sprintf("%s-upgrader-job", delegateName)
	_, err := k8s.RunKubectlAndGetOutputE(t, kubectlOptions, "get", "cronjob", cronjobName)
	require.NoError(t, err, "CronJob %s does not exist", cronjobName)
}

// // ValidateMTLSVolumeConfiguration validates that mTLS volumes are properly configured
// func ValidateMTLSVolumeConfiguration(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName, expectedSecretName string) {
// 	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)

// 	// Check for volumes in the pod template
// 	volumes := deployment.Spec.Template.Spec.Volumes
// 	var mtlsVolume *corev1.Volume
// 	for _, vol := range volumes {
// 		if vol.Secret != nil && vol.Secret.SecretName == expectedSecretName {
// 			mtlsVolume = &vol
// 			break
// 		}
// 	}

// 	require.NotNil(t, mtlsVolume, "mTLS volume should exist")
// 	require.Equal(t, expectedSecretName, mtlsVolume.Secret.SecretName, "Volume should reference the correct mTLS secret")

// 	// Check for volume mounts
// 	containers := deployment.Spec.Template.Spec.Containers
// 	require.Greater(t, len(containers), 0, "Deployment should have at least one container")

// 	volumeMounts := containers[0].VolumeMounts
// 	var mtlsVolumeMount *corev1.VolumeMount
// 	for _, vm := range volumeMounts {
// 		if strings.Contains(vm.Name, "mtls") || strings.Contains(vm.Name, "tls") {
// 			mtlsVolumeMount = &vm
// 			break
// 		}
// 	}

// 	if mtlsVolumeMount != nil {
// 		require.NotEmpty(t, mtlsVolumeMount.MountPath, "mTLS volume should have a mount path")
// 		require.True(t, mtlsVolumeMount.ReadOnly, "mTLS volume should be read-only")
// 	}
// }

// // ValidateNoMTLSConfiguration validates that no mTLS volumes are configured
// func ValidateNoMTLSConfiguration(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string) {
// 	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)

// 	containers := deployment.Spec.Template.Spec.Containers
// 	require.Greater(t, len(containers), 0, "Deployment should have at least one container")

// 	// Check that no mTLS-related volume mounts exist
// 	volumeMounts := containers[0].VolumeMounts
// 	for _, vm := range volumeMounts {
// 		require.False(t, strings.Contains(strings.ToLower(vm.Name), "mtls"),
// 			fmt.Sprintf("Should not have mTLS volume mount: %s", vm.Name))
// 		require.False(t, strings.Contains(strings.ToLower(vm.Name), "tls"),
// 			fmt.Sprintf("Should not have TLS volume mount: %s", vm.Name))
// 	}

// 	// Check that no mTLS-related volumes exist
// 	volumes := deployment.Spec.Template.Spec.Volumes
// 	for _, vol := range volumes {
// 		if vol.Secret != nil {
// 			require.False(t, strings.Contains(strings.ToLower(vol.Secret.SecretName), "mtls"),
// 				fmt.Sprintf("Should not have mTLS secret volume: %s", vol.Secret.SecretName))
// 		}
// 	}
// }

// ProxyConfig represents proxy configuration for testing
type ProxyConfig struct {
	Host     string
	Port     string
	Scheme   string
	User     string
	Password string
	NoProxy  string
}

// // CleanupSecrets performs cleanup of test secrets
// func CleanupSecrets(t *testing.T, kubectlOptions *k8s.KubectlOptions, additionalSecrets []string) {
// 	// Clean up any additional secrets created manually
// 	for _, secretName := range additionalSecrets {
// 		if secretName != "" {
// 			secret := map[string]interface{}{
// 				"apiVersion": "v1",
// 				"kind":       "Secret",
// 				"metadata": map[string]interface{}{
// 					"name":      secretName,
// 					"namespace": kubectlOptions.Namespace,
// 				},
// 			}
// 			yamlBytes, _ := yaml.Marshal(secret)
// 			k8s.KubectlDeleteFromStringE(t, kubectlOptions, string(yamlBytes))
// 		}
// 	}
// }

// ValidateHelmRelease validates that the Helm release is properly deployed
func ValidateHelmRelease(t *testing.T, kubectlOptions *k8s.KubectlOptions, namespaceName, delegateName string) {
	helmOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
	}

	releases := ListHelmReleases(t, helmOptions, namespaceName)
	var foundRelease bool

	for _, release := range releases {
		if release.Name == delegateName && release.Namespace == namespaceName {
			foundRelease = true
			assert.Equal(t, "deployed", release.Status)
			break
		}
	}
	require.True(t, foundRelease, "Helm release should exist and be deployed")
}

// ResolveContainerEnvMap returns map[name]=value for a container by resolving:
// - env.Value
// - env.ValueFrom.{ConfigMapKeyRef, SecretKeyRef}
// - envFrom.{ConfigMapRef, SecretRef} (imports all keys)
// Also checks for requierd ConfigMapKeyRef and SecretKeyRef
func ResolveContainerEnvMap(t *testing.T, options *k8s.KubectlOptions, container corev1.Container) map[string]string {
	result := make(map[string]string)

	// 1) explicit env vars
	for _, e := range container.Env {
		if e.Value != "" {
			result[e.Name] = e.Value
			continue
		}
		if e.ValueFrom == nil {
			continue
		}

		// ConfigMapKeyRef
		if cmRef := e.ValueFrom.ConfigMapKeyRef; cmRef != nil {
			cm, err := k8s.GetConfigMapE(t, options, cmRef.Name)
			if err != nil {
				require.NoError(t, err, "ConfigMap %s not found for env %s", cmRef.Name, e.Name)
			} else if v, ok := cm.Data[cmRef.Key]; ok {
				result[e.Name] = v
			}
		}

		// SecretKeyRef
		if secRef := e.ValueFrom.SecretKeyRef; secRef != nil {
			secret, err := k8s.GetSecretE(t, options, secRef.Name)
			if err != nil {
				require.NoError(t, err, "Secret %s not found for env %s", secRef.Name, e.Name)
			} else if v, ok := secret.Data[secRef.Key]; ok {
				result[e.Name] = string(v)
			}
		}
	}

	// 2) envFrom (bulk import)
	for _, ef := range container.EnvFrom {
		if ef.ConfigMapRef != nil && ef.ConfigMapRef.Name != "" {
			cm, err := k8s.GetConfigMapE(t, options, ef.ConfigMapRef.Name)
			if err != nil {
				if ef.ConfigMapRef.Optional != nil && *ef.ConfigMapRef.Optional {
					t.Logf("optional ConfigMap %s not found, skipping", ef.ConfigMapRef.Name)
					continue
				}
				require.NoError(t, err, "ConfigMap %s not found", ef.ConfigMapRef.Name)
			}
			for k, v := range cm.Data {
				if _, exists := result[k]; !exists {
					result[k] = v
				}
			}
		}
		if ef.SecretRef != nil && ef.SecretRef.Name != "" {
			secret, err := k8s.GetSecretE(t, options, ef.SecretRef.Name)
			if err != nil {
				if ef.SecretRef.Optional != nil && *ef.SecretRef.Optional {
					t.Logf("optional Secret %s not found, skipping", ef.SecretRef.Name)
					continue
				}
				require.NoError(t, err, "Secret %s not found", ef.SecretRef.Name)
			}
			for k, b := range secret.Data {
				if _, exists := result[k]; !exists {
					result[k] = string(b)
				}
			}
		}
	}

	return result
}
