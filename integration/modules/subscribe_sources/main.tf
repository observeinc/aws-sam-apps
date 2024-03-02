resource "aws_sns_topic_subscription" "sns" {
  protocol  = "sqs"
  topic_arn = var.sources.sns_topic.arn
  endpoint  = var.queue_arn
}

resource "aws_s3_bucket_notification" "sqs" {
  bucket = var.sources.buckets.sqs.id
  queue {
    queue_arn = var.queue_arn
    events    = ["s3:ObjectCreated:*"]
  }
}

resource "aws_s3_bucket_notification" "kms" {
  bucket = var.sources.buckets.kms.id
  queue {
    queue_arn = var.queue_arn
    events    = ["s3:ObjectCreated:*"]
  }
}

resource "aws_s3_bucket_notification" "eventbridge" {
  bucket      = var.sources.buckets.eventbridge.id
  eventbridge = true
}
