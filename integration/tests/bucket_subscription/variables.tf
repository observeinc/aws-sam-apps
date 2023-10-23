variable "bucket" {
  type = string
}

variable "queue_arn" {
  type    = string
  default = null
}

variable "topic_arn" {
  type    = string
  default = null
}

variable "eventbridge" {
  type    = bool
  default = false
}
