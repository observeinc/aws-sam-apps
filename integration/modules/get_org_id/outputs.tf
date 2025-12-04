output "org_id" {
  description = "AWS Organizations ID"
  value       = data.aws_organizations_organization.current.id
}

output "account_id" {
  description = "AWS Account ID"
  value       = data.aws_caller_identity.current.account_id
}

