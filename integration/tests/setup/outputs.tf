output "id" {
  value       = random_pet.run.id
  description = "Random test identifier"
}

output "source" {
  value       = aws_s3_access_point.source
  description = "S3 bucket where files are copied from"
}

output "destination" {
  value       = aws_s3_bucket.destination
  description = "S3 bucket where files are copied to"
}
