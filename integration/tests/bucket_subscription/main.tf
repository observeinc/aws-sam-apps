resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = var.bucket

  dynamic "queue" {
    for_each = var.queue_arn != null ? [1] : []
    content {
      queue_arn = var.queue_arn
      events    = ["s3:ObjectCreated:*"]
    }
  }

  dynamic "topic" {
    for_each = var.topic_arn != null ? [1] : []
    content {
      topic_arn = var.topic_arn
      events    = ["s3:ObjectCreated:*"]
    }
  }

  eventbridge = var.eventbridge
}
