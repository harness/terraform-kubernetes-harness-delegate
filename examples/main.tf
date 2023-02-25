module "delegate" {
  source = "harness/harness-delegate/kubernetes"
  version = "0.1.5"

  account_id = "PUT_YOUR_ACCOUNT_ID"
  delegate_token = "PUT_YOUR_DELEGATE_TOKEN"
  delegate_name = "PUT_YOUR_DELEGATE_NAME"
  namespace = "harness-delegate-ng"
  manager_endpoint = "PUT_YOUR_MANAGER_URL"
  delegate_image = "PUT_YOUR_DELEGATE_IMAGE"
  replicas = 1
  upgrader_enabled = false

  # Additional optional values to pass to the helm chart
  values = yamlencode({
    initScript: ""
  })
}

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}
