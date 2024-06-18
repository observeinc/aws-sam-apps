# This test verifies our forwarder can write to an S3 bucket directly,
# without being fronted by a DataAccessPoint
run "setup" {
  module {
    source  = "observeinc/collection/aws//modules/testing/setup"
    version = "2.9.0"
  }
}

run "target_bucket" {
  module {
    source  = "observeinc/collection/aws//modules/testing/s3_bucket"
    version = "2.9.0"
  }

  variables {
    setup = run.setup
  }
}

run "sources" {
  module {
    source = "./modules/setup_sources"
  }

  variables {
    setup = run.setup
  }
}

run "install_forwarder" {
  variables {
    setup = run.setup
    app   = "forwarder"
    parameters = {
      DestinationUri    = "s3://${run.target_bucket.id}/"
      SourceBucketNames = "${join(",", [for k, v in run.sources.buckets : v.id])}"
      SourceObjectKeys  = "*/allowed/*"
      SourceTopicArns   = "arn:aws:sns:${run.setup.region}:${run.setup.account_id}:*"
      NameOverride      = run.setup.id
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "subscribe_sources" {
  module {
    source = "./modules/subscribe_sources"
  }

  variables {
    sources   = run.sources
    queue_arn = run.install_forwarder.stack.outputs["QueueArn"]
  }
}


run "check_sqs" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE        = run.sources.buckets["sqs"].id
      DESTINATION   = run.target_bucket.id
      OBJECT_PREFIX = "test/allowed/"
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to copy object using SQS"
  }
}

run "check_disallowed" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.sources.buckets["sqs"].id
      DESTINATION = run.target_bucket.id
    }
  }

  assert {
    condition     = output.exitcode != 0
    error_message = "Succeeded copying object not in source object keys"
  }
}
