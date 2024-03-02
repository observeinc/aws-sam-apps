locals {
  sources = toset(["sns", "sqs", "eventbridge"])
}

module "buckets" {
  for_each            = local.sources
  source              = "observeinc/collection/aws//modules/testing/s3_bucket"
  version             = "2.9.0"
  setup               = var.setup
  enable_access_point = false
}

resource "aws_sns_topic" "this" {
  name = var.setup.short
}

data "aws_iam_policy_document" "s3_to_sns" {
  statement {
    actions   = ["SNS:Publish"]
    resources = [aws_sns_topic.this.arn]
    principals {
      type        = "Service"
      identifiers = ["s3.amazonaws.com"]
    }
  }
}

resource "aws_sns_topic_policy" "s3_to_sns" {
  arn    = aws_sns_topic.this.arn
  policy = data.aws_iam_policy_document.s3_to_sns.json
}

resource "aws_s3_bucket_notification" "sns" {
  bucket = module.buckets["sns"].id
  topic {
    topic_arn = aws_sns_topic.this.arn
    events    = ["s3:ObjectCreated:*"]
  }

  depends_on = [aws_sns_topic_policy.s3_to_sns]
}
