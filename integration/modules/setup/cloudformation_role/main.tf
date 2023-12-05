resource "random_pet" "run" {
  length = 2
}

resource "aws_iam_role" "cloudformation_role" {
  name = "${var.stack_name}-${random_pet.run.id}"

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

resource "aws_iam_role_policy" "example_policy" {
  name   = "${var.stack_name}-${random_pet.run.id}_policy"
  role   = aws_iam_role.cloudformation_role.id
  policy = file("${path.module}/policy_${var.stack_name}.json")
}
