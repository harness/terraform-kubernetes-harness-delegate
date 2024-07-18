terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~2"
    }
    utils = {
      source  = "cloudposse/utils"
      version = ">= 0.14.0"
    }
  }
}
