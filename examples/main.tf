module "delegate" {
  source = "harness/kubernetes-delegate/harness"
  version = "0.1.5"

  account_id = ""
  delegate_token = ""
  delegate_name = "example"
  namespace = "harness-delegate-ng"
  manager_endpoint = "https://app.harness.io/gratis"
  # delegate_image = "harness/delegate:22.12.77802"
  replicas = 1
  upgrader_enabled = false

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
