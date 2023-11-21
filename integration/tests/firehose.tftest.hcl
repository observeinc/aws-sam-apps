run "setup" {
  module {
    source = "./modules/setup/run"
  }
}

run "install" {
  variables {
    name = run.setup.id
    app  = "firehose"
    parameters = {
      BucketARN = "arn:aws:s3:::${run.setup.access_point.bucket}"
    }
    capabilities = [
      "CAPABILITY_IAM",
    ]
  }
}

run "check_firehose" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_firehose"
    env_vars = {
      FIREHOSE_ARN = run.install.stack.outputs["Firehose"]
      DESTINATION  = "s3://${run.setup.access_point.bucket}/"
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to write firehose records"
  }
}

run "set_prefix" {
  variables {
    name = run.setup.id
    app  = "firehose"
    parameters = {
      BucketARN         = "arn:aws:s3:::${run.setup.access_point.bucket}"
      Prefix            = "${run.setup.id}/"
      WriterRoleService = "logs.amazonaws.com"
    }
    capabilities = [
      "CAPABILITY_IAM",
    ]
  }
}

run "check_firehose_prefix" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_firehose"
    env_vars = {
      FIREHOSE_ARN = run.install.stack.outputs["Firehose"]
      DESTINATION  = "s3://${run.setup.access_point.bucket}/${run.setup.id}/"
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to write firehose records"
  }
}
