data "aws_region" "current" {}

resource "aws_cloudformation_stack" "this" {
  name          = var.name
  template_body = file("../.aws-sam/build/${var.app}/${data.aws_region.current.name}/packaged.yaml")
  parameters    = var.parameters
  capabilities  = var.capabilities
  iam_role_arn  = var.cloudformation_role
}
