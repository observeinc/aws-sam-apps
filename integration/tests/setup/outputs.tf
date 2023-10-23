output "id" {
  value       = random_pet.run.id
  description = "Random test identifier"
}

output "source_bucket" {
  value       = aws_s3_bucket.source.bucket
  description = "S3 bucket where files are copied from"
}

output "destination_arn" {
  value       = aws_s3_access_point.destination.arn
  description = "S3 access point ARN where files are copied to"
}

output "queue_arn" {
  value       = aws_cloudformation_stack.this.outputs["Queue"]
  description = "ARN of the SQS Queue from the CloudFormation stack"
}

output "stack" {
  value = aws_cloudformation_stack.this
}
