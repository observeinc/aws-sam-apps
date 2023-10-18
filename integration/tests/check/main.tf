data "external" "check" {
  program = concat(["${path.module}/run"], var.program)
  query   = var.env_vars
}
