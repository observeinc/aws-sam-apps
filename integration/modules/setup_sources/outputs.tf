output "buckets" {
  description = "Buckets by source"
  value       = module.buckets
}

output "sns_topic" {
  description = "SNS Topic which SNS source bucket is subscribed to"
  value       = aws_sns_topic.this
}
