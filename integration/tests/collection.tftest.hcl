run "setup" {
  module {
    source = "./modules/setup/run"
  }
}

run "install_collection" {
  variables {
    name        = "collection-stack-${run.setup.id}"
    app         = "collection"
    parameters  = {
      DataAccessPointArn   = run.setup.access_point.arn
      DestinationUri       = "s3://${run.setup.access_point.alias}"
      LogGroupNamePatterns = "*"
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "check_sqs" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.install_collection.stack.outputs["Bucket"]
      DESTINATION = run.setup.access_point.bucket
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to copy object using SQS"
  }
}
