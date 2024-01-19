run "setup" {
  module {
    source  = "observeinc/collection/aws//modules/testing/run"
    version = "2.6.0"
  }
}

run "check" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.6.0"
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
