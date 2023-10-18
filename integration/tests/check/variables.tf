variable "program" {
  description = <<-EOF
    A list of strings, whose first element is the program to run and whose
    subsequent elements are optional command line arguments to the program.
    Terraform does not execute the program through a shell, so it is not
    necessary to escape shell metacharacters nor add quotes around arguments
    containing spaces.
  EOF
  type        = list(string)
  nullable    = false
}

variable "env_vars" {
  description = <<-EOF
    Environment variables
  EOF
  type        = map(string)
  default     = {}
  nullable    = false
}
