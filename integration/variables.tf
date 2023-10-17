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


