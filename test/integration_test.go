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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestDelegateIntegrationScenarios tests various realistic deployment scenarios
func TestDelegateIntegrationScenarios(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		description string
		vars        map[string]interface{}
		setupFunc   func(t *testing.T, kubectlOptions *k8s.KubectlOptions) []string // Returns list of secrets to cleanup
		validateFunc func(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, terraformOptions *terraform.Options)
	}{
		{
			name:        "production_like_deployment",
			description: "Production-like deployment with multiple replicas and upgrader enabled",
			vars: map[string]interface{}{
				"replicas":         3,
				"upgrader_enabled": true,
				"deploy_mode":      "KUBERNETES",
				"next_gen":         true,
			},
			setupFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions) []string {
				return []string{} // No additional setup needed
			},
			validateFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, terraformOptions *terraform.Options) {
				// Validate multiple replicas
				deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
				assert.Equal(t, int32(3), *deployment.Spec.Replicas)
				assert.Equal(t, int32(3), deployment.Status.ReadyReplicas)
				
				// Validate upgrader is enabled
				envMap := GetEnvironmentVariables(t, kubectlOptions, delegateName)
				assert.Equal(t, "true", envMap["ENABLE_UPGRADER"])
			},
		},
		{
			name:        "corporate_proxy_deployment",
			description: "Deployment behind corporate proxy with authentication",
			vars: map[string]interface{}{
				"proxy_host":     "corporate-proxy.company.com",
				"proxy_port":     "3128",
				"proxy_scheme":   "https",
				"proxy_user":     "delegate_service_account",
				"proxy_password": "secure_proxy_password",
				"no_proxy":       ".company.com,.internal,localhost,127.0.0.1",
			},
			setupFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions) []string {
				return []string{} // No additional setup needed
			},
			validateFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, terraformOptions *terraform.Options) {
				envMap := GetEnvironmentVariables(t, kubectlOptions, delegateName)
				expectedProxy := ProxyConfig{
					Host:     "corporate-proxy.company.com",
					Port:     "3128",
					Scheme:   "https",
					User:     "delegate_service_account",
					Password: "secure_proxy_password",
					NoProxy:  ".company.com,.internal,localhost,127.0.0.1",
				}
				ValidateProxyConfiguration(t, envMap, expectedProxy)
			},
		},
		{
			name:        "secure_mtls_deployment",
			description: "Deployment with mTLS configuration for secure communication",
			vars: map[string]interface{}{
				"replicas": 2,
			},
			setupFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions) []string {
				// Create a realistic mTLS secret with proper certificate structure
				secretName := fmt.Sprintf("harness-delegate-mtls-%s", strings.ToLower(random.UniqueId()))
				
				// Simulate more realistic certificate data
				certData := `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKoK/heBjcOuMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTIwOTEyMjE1MjAyWhcNMTcwOTExMjE1MjAyWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAwuqTaN4dJRiKUpHQ=
-----END CERTIFICATE-----`
				
				keyData := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDC6pNo3h0lGIpS
kdDUPFgYKg0jtWKOL7q+9qZsDKfKgEzGNkrpJQj6vLCWGd/M2F5mZGx9dzFM2dE+
/YjHGp6C3eEQRzF2q9V9k9r0QjWcHrGlJAM=
-----END PRIVATE KEY-----`
				
				caData := `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKoK/heBjcOuMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTIwOTEyMjE1MjAyWhcNMTcwOTExMjE1MjAyWjBF
-----END CERTIFICATE-----`
				
				data := map[string][]byte{
					"tls.crt": []byte(certData),
					"tls.key": []byte(keyData),
					"ca.crt":  []byte(caData),
				}
				
				CreateTestSecret(t, kubectlOptions, secretName, string(corev1.SecretTypeTLS), data)
				return []string{secretName}
			},
			validateFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, terraformOptions *terraform.Options) {
				// Validate deployment has 2 replicas
				deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
				assert.Equal(t, int32(2), *deployment.Spec.Replicas)
				assert.Equal(t, int32(2), deployment.Status.ReadyReplicas)
				
				// Validate mTLS secret is properly referenced
				mtlsSecretName := terraformOptions.Vars["mtls_secret_name"].(string)
				ValidateMTLSVolumeConfiguration(t, kubectlOptions, delegateName, mtlsSecretName)
				
				// Verify the secret exists and contains proper data
				secret := k8s.GetSecret(t, kubectlOptions, mtlsSecretName)
				assert.Contains(t, string(secret.Data["tls.crt"]), "BEGIN CERTIFICATE")
				assert.Contains(t, string(secret.Data["tls.key"]), "BEGIN PRIVATE KEY")
				assert.Contains(t, string(secret.Data["ca.crt"]), "BEGIN CERTIFICATE")
			},
		},
		{
			name:        "onprem_deployment",
			description: "On-premises deployment configuration",
			vars: map[string]interface{}{
				"deploy_mode": "KUBERNETES_ONPREM",
				"next_gen":    false, // Test first gen delegate
				"replicas":    1,
			},
			setupFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions) []string {
				return []string{} // No additional setup needed
			},
			validateFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, terraformOptions *terraform.Options) {
				envMap := GetEnvironmentVariables(t, kubectlOptions, delegateName)
				assert.Equal(t, "KUBERNETES_ONPREM", envMap["DEPLOY_MODE"])
				assert.Equal(t, "false", envMap["NEXT_GEN"])
			},
		},
		{
			name:        "custom_values_deployment",
			description: "Deployment with custom Helm values",
			vars: map[string]interface{}{
				"values": `
resources:
  limits:
    cpu: "1000m"
    memory: "2Gi"
  requests:
    cpu: "500m"
    memory: "1Gi"
nodeSelector:
  disktype: ssd
tolerations:
  - key: "dedicated"
    operator: "Equal"
    value: "harness"
    effect: "NoSchedule"
`,
			},
			setupFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions) []string {
				return []string{} // No additional setup needed
			},
			validateFunc: func(t *testing.T, kubectlOptions *k8s.KubectlOptions, delegateName string, terraformOptions *terraform.Options) {
				deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
				
				// Validate custom resource limits are applied
				containers := deployment.Spec.Template.Spec.Containers
				if len(containers) > 0 && containers[0].Resources.Limits != nil {
					cpuLimit := containers[0].Resources.Limits["cpu"]
					memoryLimit := containers[0].Resources.Limits["memory"]
					if !cpuLimit.IsZero() {
						assert.Equal(t, "1", cpuLimit.String())
					}
					if !memoryLimit.IsZero() {
						assert.Equal(t, "2Gi", memoryLimit.String())
					}
				}
				
				// Validate node selector
				nodeSelector := deployment.Spec.Template.Spec.NodeSelector
				if nodeSelector != nil {
					assert.Equal(t, "ssd", nodeSelector["disktype"])
				}
				
				// Validate tolerations
				tolerations := deployment.Spec.Template.Spec.Tolerations
				if len(tolerations) > 0 {
					found := false
					for _, toleration := range tolerations {
						if toleration.Key == "dedicated" && toleration.Value == "harness" {
							found = true
							break
						}
					}
					assert.True(t, found, "Should have dedicated=harness toleration")
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			// Generate unique resource names
			uniqueID := random.UniqueId()
			namespaceName, delegateName := GenerateUniqueTestNames(fmt.Sprintf("test-%s", tc.name), uniqueID)
			
			// Setup kubectl options
			kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
			
			// Create namespace manually if needed for setup
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			k8s.CreateNamespace(t, kubectlOptions, namespace)
			
			// Run setup function and get secrets to cleanup
			secretsToCleanup := tc.setupFunc(t, kubectlOptions)
			
			// Merge test-specific vars with defaults
			vars := DefaultTerraformVars(namespaceName, delegateName)
			for k, v := range tc.vars {
				vars[k] = v
			}
			
			// Add mTLS secret name if secrets were created
			if len(secretsToCleanup) > 0 {
				vars["mtls_secret_name"] = secretsToCleanup[0]
			}
			
			// Don't create namespace via Terraform since we created it manually
			vars["create_namespace"] = false
			
			// Create terraform options
			terraformOptions := CreateTerraformOptions(t, vars)
			
			// Setup cleanup
			defer func() {
				CleanupResources(t, terraformOptions, kubectlOptions, secretsToCleanup)
			}()
			
			// Run terraform
			terraform.InitAndApply(t, terraformOptions)
			
			// Wait for deployment to be ready
			WaitForDelegateDeployment(t, kubectlOptions, delegateName, 45*time.Second)
			
			// Run test-specific validation
			tc.validateFunc(t, kubectlOptions, delegateName, terraformOptions)
			
			// Common validations for all scenarios
			ValidateHelmRelease(t, kubectlOptions, delegateName)
			
			// Validate terraform output
			output := terraform.Output(t, terraformOptions, "values")
			assert.NotEmpty(t, output, "Terraform output should not be empty")
		})
	}
}

// TestDelegateUpgradeScenario tests the upgrade scenario
func TestDelegateUpgradeScenario(t *testing.T) {
	t.Parallel()
	
	// Generate unique resource names
	uniqueID := random.UniqueId()
	namespaceName, delegateName := GenerateUniqueTestNames("test-upgrade", uniqueID)
	
	// First deployment with upgrader disabled
	initialVars := DefaultTerraformVars(namespaceName, delegateName)
	initialVars["upgrader_enabled"] = false
	initialVars["replicas"] = 1
	
	terraformOptions := CreateTerraformOptions(t, initialVars)
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	
	defer CleanupResources(t, terraformOptions, kubectlOptions, []string{})
	
	// Initial deployment
	terraform.InitAndApply(t, terraformOptions)
	WaitForDelegateDeployment(t, kubectlOptions, delegateName, 30*time.Second)
	
	// Verify initial state
	deployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	assert.Equal(t, int32(1), *deployment.Spec.Replicas)
	
	envMap := GetEnvironmentVariables(t, kubectlOptions, delegateName)
	assert.Equal(t, "false", envMap["ENABLE_UPGRADER"])
	
	// Update configuration to enable upgrader and increase replicas
	upgradedVars := DefaultTerraformVars(namespaceName, delegateName)
	upgradedVars["upgrader_enabled"] = true
	upgradedVars["replicas"] = 2
	upgradedVars["create_namespace"] = false // Namespace already exists
	
	terraformOptionsUpgraded := CreateTerraformOptions(t, upgradedVars)
	
	// Apply upgrade
	terraform.Apply(t, terraformOptionsUpgraded)
	
	// Wait for upgraded deployment
	WaitForDelegateDeployment(t, kubectlOptions, delegateName, 45*time.Second)
	
	// Verify upgraded state
	upgradedDeployment := k8s.GetDeployment(t, kubectlOptions, delegateName)
	assert.Equal(t, int32(2), *upgradedDeployment.Spec.Replicas)
	assert.Equal(t, int32(2), upgradedDeployment.Status.ReadyReplicas)
	
	upgradedEnvMap := GetEnvironmentVariables(t, kubectlOptions, delegateName)
	assert.Equal(t, "true", upgradedEnvMap["ENABLE_UPGRADER"])
}