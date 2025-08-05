<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_helm"></a> [helm](#requirement\_helm) | 2.9.0 |
| <a name="requirement_utils"></a> [utils](#requirement\_utils) | >= 0.14.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_helm"></a> [helm](#provider\_helm) | 2.9.0 |
| <a name="provider_utils"></a> [utils](#provider\_utils) | >= 0.14.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [helm_release.delegate](https://registry.terraform.io/providers/hashicorp/helm/2.9.0/docs/resources/release) | resource |
| [utils_deep_merge_yaml.values](https://registry.terraform.io/providers/cloudposse/utils/latest/docs/data-sources/deep_merge_yaml) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_account_id"></a> [account\_id](#input\_account\_id) | The account ID to use for the Harness delegate. | `string` | n/a | yes |
| <a name="input_create_namespace"></a> [create\_namespace](#input\_create\_namespace) | Create namespace if it does not exist | `bool` | `true` | no |
| <a name="input_delegate_image"></a> [delegate\_image](#input\_delegate\_image) | The image of delegate. | `string` | `""` | no |
| <a name="input_delegate_name"></a> [delegate\_name](#input\_delegate\_name) | The name of the Harness delegate. | `string` | n/a | yes |
| <a name="input_delegate_token"></a> [delegate\_token](#input\_delegate\_token) | The account secret to use for the Harness delegate. | `string` | n/a | yes |
| <a name="input_deploy_mode"></a> [deploy\_mode](#input\_deploy\_mode) | Delegate deploy\_mode, options are 'KUBERNETES', 'KUBERNETES\_ONPREM', 'ONPREM'. | `string` | `"KUBERNETES"` | no |
| <a name="input_helm_repository"></a> [helm\_repository](#input\_helm\_repository) | The Helm repository to use. | `string` | `"https://app.harness.io/storage/harness-download/delegate-helm-chart/"` | no |
| <a name="input_init_script"></a> [init\_script](#input\_init\_script) | Init Script | `string` | `""` | no |
| <a name="input_manager_endpoint"></a> [manager\_endpoint](#input\_manager\_endpoint) | The endpoint of Harness Manager. | `string` | n/a | yes |
| <a name="input_namespace"></a> [namespace](#input\_namespace) | The namespace to deploy the Harness delegate to. | `string` | `"harness-delegate-ng"` | no |
| <a name="input_next_gen"></a> [next\_gen](#input\_next\_gen) | Is next gen or first gen delegate. | `bool` | `true` | no |
| <a name="input_no_proxy"></a> [no\_proxy](#input\_no\_proxy) | Enter a comma-separated list of suffixes that do not need the proxy. For example, .company.com,hostname,etc. Do not use leading wildcards. | `string` | `""` | no |
| <a name="input_proxy_host"></a> [proxy\_host](#input\_proxy\_host) | The proxy host. | `string` | `""` | no |
| <a name="input_proxy_password"></a> [proxy\_password](#input\_proxy\_password) | The proxy password to use for the Harness delegate. | `string` | `""` | no |
| <a name="input_proxy_port"></a> [proxy\_port](#input\_proxy\_port) | The port of the proxy | `string` | `""` | no |
| <a name="input_proxy_scheme"></a> [proxy\_scheme](#input\_proxy\_scheme) | The proxy user to use for the Harness delegate. | `string` | `""` | no |
| <a name="input_proxy_user"></a> [proxy\_user](#input\_proxy\_user) | The proxy user to use for the Harness delegate. | `string` | `""` | no |
| <a name="input_replicas"></a> [replicas](#input\_replicas) | replica count of delegates. | `number` | `1` | no |
| <a name="input_upgrader_enabled"></a> [upgrader\_enabled](#input\_upgrader\_enabled) | Is upgrader enabled | `bool` | `true` | no |
| <a name="input_mtls_secret_name"></a> [mtls\_secret\_name](#input\_mtls\_secret\_name) | The name of the mTLS secret. | `string` | `""` | no |
| <a name="input_values"></a> [values](#input\_values) | Additional values to pass to the helm chart. Values will be merged, in order, as Helm does with multiple -f options | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_values"></a> [values](#output\_values) | n/a |
<!-- END_TF_DOCS -->