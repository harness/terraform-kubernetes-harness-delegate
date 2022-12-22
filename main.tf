resource "helm_release" "delegate" {
  name             = var.delegate_name
  repository       = var.helm_repository
  chart            = "harness-delegate-ng"
  namespace        = var.namespace
  create_namespace = true

  values = [data.utils_deep_merge_yaml.values.output]
}

locals {
  values = yamlencode({
    accountId            = var.account_id,
    delegateToken        = var.delegate_token,
    managerEndpoint      = var.manager_endpoint,
    namespace            = var.namespace,
    delegateName         = var.delegate_name,
    delegateDockerImage  = var.delegate_image,
    replicas             = var.replicas,
    upgrader             = { enabled = var.upgrader_enabled }
    proxyUser            = var.proxy_user,
    proxyPassword        = var.proxy_password,
    proxyHost            = var.proxy_host,
    proxyPort            = var.proxy_port,
    proxyScheme          = var.proxy_scheme,
    noProxy              = var.no_proxy
  })
}

data "utils_deep_merge_yaml" "values" {
  input = compact([
    local.values,
    var.values
  ])
}

output "values" {
  value = data.utils_deep_merge_yaml.values.output
  // sensitive = false
}
