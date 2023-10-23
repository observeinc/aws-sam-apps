data "external" "check" {
  program = concat(["${path.module}/run"], [var.command], var.args)
  query   = var.env_vars
}
