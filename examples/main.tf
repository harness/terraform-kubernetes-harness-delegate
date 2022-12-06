module "delegate" {
  source = "harness/kubernetes-delegate/harness"
  version = "0.1.1"

  account_id = "ABC123"
  delegate_token = "XYZ789"
  delegate_name = "example"
  namespace = "harness-delegate-ng"
  manager_endpoint = "https://app.harness.io/gratis"

  # Additional optional values to pass to the helm chart
  values = yamlencode({
    javaOpts: "-Xms64M" 
  })
}

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}
