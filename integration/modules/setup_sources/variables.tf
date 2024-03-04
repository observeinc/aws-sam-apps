variable "setup" {
  type = object({
    short = string
  })
  description = "Setup module."
}

variable "queue_arn" {
  description = "Queue ARN"
  type        = string
  default     = null
}
