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
    }
  }

  assert {
    condition     = output.exitcode != 0
    error_message = "Bucket isn't empty"
  }
}
