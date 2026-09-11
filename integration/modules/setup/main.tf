locals {
  prefix        = "integration-test-"
  prefix_length = length(local.prefix)
}

module "upstream" {
  source  = "observeinc/collection/aws//modules/testing/setup"
  version = "2.9.0"

  id_length    = var.id_length - local.prefix_length
  short_length = var.short_length - local.prefix_length
}
