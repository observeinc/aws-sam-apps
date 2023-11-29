run "setup" {
  module {
    source = "./modules/setup/run"
  }
}

run "check" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_bucket_not_empty"
    env_vars = {
      SOURCE = run.setup.access_point.bucket
      OPTS   = "--output json"
    }
  }

  assert {
    condition     = output.error == "bucket is empty"
    error_message = "Bucket isn't empty"
  }
}
