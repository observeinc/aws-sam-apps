variable "name" {
  description = "Stack name"
  type        = string
}

variable "app" {
  description = "App name"
  type        = string
}

variable "parameters" {
  description = "Stack parameters"
  type        = map(string)
}

variable "capabilities" {
  description = "Stack capabilities"
  type        = list(string)
}

variable "install_policy_json" {
  description = "Cloudformation policy to associate to role used for install."
  type        = string
  default     = null
  validation {
    condition     = can(jsondecode(var.install_policy_json))
    error_message = "must be valid JSON"
  }
}

variable "tags" {
  description = <<-EOF
    Set of tags to apply to stack. By default we always apply at least one tag
    in order to exercise the required IAM policies.
  EOF
  type        = map(string)
  default     = { hello = "world" }
}
