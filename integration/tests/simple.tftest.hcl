run "setup" {
  module {
    source = "./tests/setup"
  }
}

run "check" {
  module {
    source = "./tests/check"
  }

  variables {
    program = ["./scripts/check_bucket_not_empty"]
    env_vars = {
      SOURCE = run.setup.source.bucket
    }
  }

  assert {
    condition     = output.result.error == ""
    error_message = "Bucket not empty check failed"
  }
}
