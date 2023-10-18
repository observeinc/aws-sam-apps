run "setup" {
  module {
    source = "./tests/setup"
  }
}

run "install_forwarder" {
  variables {
    name = run.setup.id
    app = "forwarder"
    parameters = {
      DataAccessPointArn = run.setup.destination.arn
      DestinationUri     = "s3://${run.setup.destination.alias}"
      SourceBucketNames  = run.setup.source.bucket
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "check_file_not_copied" {
  module {
    source = "./tests/check"
  }

  variables {
    command = "./scripts/check_object_copy"
    env_vars = {
      SOURCE      = run.setup.source.bucket
      DESTINATION = run.setup.destination.bucket
    }
  }

  assert {
    condition     = output.result.error == "failed to read file from destination"
    error_message = "Unexpected error"
  }
}

run "subscribe_bucket_notifications_to_sqs" {
  module {
    source = "./tests/bucket_subscription"
  }

  variables {
    bucket    = run.setup.source.bucket
    queue_arn = run.install_forwarder.stack.outputs["Queue"]
  }
}

run "check_copy_succeeds" {
  module {
    source = "./tests/check"
  }

  variables {
    command  = "./scripts/check_object_copy"
    env_vars = {
      SOURCE      = run.setup.source.bucket
      DESTINATION = run.setup.destination.bucket
    }
  }

  assert {
    condition     = output.result.error == ""
    error_message = "Failed to copy object"
  }
}
