output "stack_name" {
  description = "Random test identifier padded out to 128 characters, prefixed with integration-test-."
  value       = trimsuffix("${local.prefix}${substr(module.upstream.stack_name, 0, 128 - local.prefix_length)}", "-")
}

output "id" {
  description = "Run identifier, prefixed with integration-test-."
  value       = trimsuffix("${local.prefix}${module.upstream.id}", "-")
}

output "short" {
  description = "Shorter identifier, prefixed with integration-test-."
  value       = trimsuffix("${local.prefix}${module.upstream.short}", "-")
}

output "region" {
  description = "AWS Region in use."
  value       = module.upstream.region
}

output "account_id" {
  description = "AWS Account in use."
  value       = module.upstream.account_id
}
