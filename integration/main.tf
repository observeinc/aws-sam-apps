data "aws_region" "current" {}

resource "aws_cloudformation_stack" "this" {
  name          = var.setup.stack_name
  template_body = file("../.aws-sam/build/regions/${data.aws_region.current.name}/${var.app}.yaml")
  parameters    = var.parameters
  capabilities  = var.capabilities
  iam_role_arn  = var.install_policy_json == null ? null : aws_iam_role.this[0].arn
  tags          = var.tags
}

resource "aws_iam_role" "this" {
  count       = var.install_policy_json == null ? 0 : 1
  name_prefix = "${var.setup.short}-"

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
}

resource "aws_iam_role_policy" "this" {
  count  = var.install_policy_json == null ? 0 : 1
  name   = "allowed"
  role   = aws_iam_role.this[0].id
  policy = var.install_policy_json
}
