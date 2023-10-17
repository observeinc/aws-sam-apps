data "external" "check" {
  program     = var.program
  query       = var.query
  working_dir = var.working_dir
}
