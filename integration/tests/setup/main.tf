resource "random_pet" "run" {
  length = 2
}

resource "aws_s3_bucket" "source" {
  bucket        = "${random_pet.run.id}-source"
  force_destroy = true
}

resource "aws_s3_bucket" "destination" {
  bucket        = "${random_pet.run.id}-destination"
  force_destroy = true
}

resource "aws_s3_access_point" "destination" {
  bucket = aws_s3_bucket.destination.id
  name   = random_pet.run.id
}

data "aws_region" "current" {}

resource "aws_cloudformation_stack" "this" {
  name          = random_pet.run.id
  template_body = file("../../../.aws-sam/build/forwarder/${data.aws_region.current.name}/packaged.yaml")
  parameters    = {
    DataAccessPointArn = aws_s3_access_point.destination.arn
    DestinationUri     = "s3://${aws_s3_access_point.destination.bucket}"
    SourceBucketNames  = aws_s3_bucket.source.bucket
  }
  capabilities  = [
    "CAPABILITY_NAMED_IAM",
    "CAPABILITY_AUTO_EXPAND",
  ]
}

resource "aws_iam_policy" "s3_put_policy" {
  name        = "S3PutPolicy"
  description = "Allows putting objects to the specified S3 bucket"
  policy      = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "s3:PutObject",
      "Resource": "${aws_s3_bucket.destination.arn}/*"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "s3_put_policy_attachment" {
  role       = element(split("/", aws_cloudformation_stack.this.outputs["Role"]), 1)
  policy_arn = aws_iam_policy.s3_put_policy.arn
}


