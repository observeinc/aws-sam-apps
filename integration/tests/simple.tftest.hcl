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
    command = "./scripts/check_bucket_not_empty"
    env_vars = {
      SOURCE = run.setup.source.bucket
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Bucket not empty check failed"
  }
}
