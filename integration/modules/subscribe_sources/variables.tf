variable "sources" {
  type = object({
    buckets = object({
      sns         = object({ id = string })
      sqs         = object({ id = string })
      eventbridge = object({ id = string })
    })
    sns_topic = object({ arn = string })
  })
  description = "Setup sources module."
  nullable    = false
}

variable "queue_arn" {
  description = "Queue ARN to subscribe to"
  type        = string
  nullable    = false
}
