variables {
  override_match        = "example"
  override_content_type = "application/x-csv;delimiter=space"
}

run "setup" {
  module {
    source = "./modules/setup/run"
  }
}

run "install_forwarder" {
  variables {
    name = run.setup.id
    app  = "forwarder"
    parameters = {
      DataAccessPointArn   = run.setup.access_point.arn
      DestinationUri       = "s3://${run.setup.access_point.alias}"
      SourceBucketNames    = "*"
      SourceTopicArn       = "arn:aws:sns:${run.setup.region}:${run.setup.account_id}:*"
      ContentTypeOverrides = "${var.override_match}=${var.override_content_type}"
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "setup_subscriptions" {
  module {
    source = "./modules/setup/subscriptions"
  }

  variables {
    run_id             = run.setup.id
    queue_arn          = run.install_forwarder.stack.outputs["Queue"]
    sources            = ["sqs", "eventbridge", "sns"]
  }
}

run "check_sqs" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.setup_subscriptions.buckets["sqs"].bucket
      DESTINATION = run.setup.access_point.bucket
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to copy object using SQS"
  }
}

run "check_eventbridge" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.setup_subscriptions.buckets["eventbridge"].bucket
      DESTINATION = run.setup.access_point.bucket
      INIT_DELAY  = 2
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to copy object using Eventbridge"
  }
}

run "check_sns" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.setup_subscriptions.buckets["sns"].bucket
      DESTINATION = run.setup.access_point.bucket
      INIT_DELAY  = 2
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to copy object using SNS"
  }
}

run "check_content_type_override" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE           = run.setup_subscriptions.buckets["sqs"].bucket
      DESTINATION      = run.setup.access_point.bucket
      # this prefix will match the content type override, so we expect the destination object
      # to have our test content type
      OBJECT_PREFIX    = var.override_match
      # modify the content type of the source to our expected value, after
      # which we should se no diff.
      JQ_PROCESS_SOURCE = ".ContentType = \"${var.override_content_type}\""
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to override content type"
  }
}
