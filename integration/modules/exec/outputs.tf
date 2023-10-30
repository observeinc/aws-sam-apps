output "error" {
  value = data.external.check.result.error
}

output "exitcode" {
  value = tonumber(data.external.check.result.exitcode)
}

output "output" {
  value = data.external.check.result.output
}
