variable "id_length" {
  type        = number
  default     = 64
  description = "Total length for id output, including the integration-test- prefix."

  validation {
    condition     = var.id_length >= 17
    error_message = "id_length must be at least 17 to accommodate the integration-test- prefix."
  }
}

variable "short_length" {
  type        = number
  default     = 32
  description = "Total length for short output, including the integration-test- prefix."

  validation {
    condition     = var.short_length >= 17
    error_message = "short_length must be at least 17 to accommodate the integration-test- prefix."
  }
}
