variable "setup" {
  type = object({
    short = string
  })
  description = "Setup module."
}

variable "kms_key_policy_json" {
  description = "JSON encoded KMS key policy. If set, the S3 bucket will be encrypted using a KMS key."
  type        = string
  default     = ""
  nullable    = false
}

variable "queue_arn" {
  description = "Queue ARN"
  type        = string
  default     = null
}
