data "aws_region" "current" {}

resource "aws_cloudformation_stack" "this" {
  name          = var.name
  template_body = file("../.aws-sam/build/${var.app}/${data.aws_region.current.name}/packaged.yaml")
  parameters    = var.parameters
  capabilities  = var.capabilities
  iam_role_arn  = var.install_policy_json == null ? null : aws_iam_role.this[0].arn
}

resource "aws_iam_role" "this" {
  count       = var.install_policy_json == null ? 0 : 1
  name_prefix = "${var.name}-"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action = "sts:AssumeRole",
        Effect = "Allow",
        Principal = {
          Service = "cloudformation.amazonaws.com"
        },
      },
    ],
  })

  inline_policy {
    name   = "allowed"
    policy = var.install_policy_json
  }
}
