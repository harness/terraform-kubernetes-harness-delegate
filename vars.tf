variable "helm_repository" {
  description = "The Helm repository to use."
  type        = string
  default     = "https://app.harness.io/storage/harness-download/delegate-helm-chart/"
}

variable "namespace" {
  description = "The namespace to deploy the Harness delegate to."
  type        = string
  default     = "harness-delegate-ng"
}

variable "delegate_image" {
  description = "The image of delegate."
  type        = string
  default     = ""
}

variable "delegate_name" {
  description = "The name of the Harness delegate."
  type        = string
  // default     = "harness-delegate"
}

variable "account_id" {
  description = "The account ID to use for the Harness delegate."
  type        = string
}

variable "delegate_token" {
  description = "The account secret to use for the Harness delegate."
  type        = string
  // sensitive = true
}

variable "manager_endpoint" {
  description = "The endpoint of Harness Manager."
  type        = string
  // default     = "https://app.harness.io/gratis"
}

variable "replicas" {
  description = "replica count of delegates."
  type        = number
  default     = 1
}

variable "upgrader_enabled" {
  description = "Is upgrader enabled"
  type        = bool
  default     = true
}

variable "proxy_user" {
  description = "The proxy user to use for the Harness delegate."
  type        = string
  // sensitive = true
  default = ""
}

variable "proxy_password" {
  description = "The proxy password to use for the Harness delegate."
  type        = string
  // sensitive = true
  default = ""
}

variable "proxy_host" {
  description = "The proxy host."
  type        = string
  // sensitive = true
  default = ""
}

variable "proxy_port" {
  description = "The port of the proxy"
  type        = string
  // sensitive = true
  default = ""
}

variable "proxy_scheme" {
  description = "The proxy user to use for the Harness delegate."
  type        = string
  // sensitive = true
  default = ""
}

variable "no_proxy" {
  description = "Enter a comma-separated list of suffixes that do not need the proxy. For example, .company.com,hostname,etc. Do not use leading wildcards."
  type        = string
  // sensitive = true
  default = ""
}

variable "values" {
  description = "Additional values to pass to the helm chart. Values will be merged, in order, as Helm does with multiple -f options"
  type        = string
  default     = ""
}
