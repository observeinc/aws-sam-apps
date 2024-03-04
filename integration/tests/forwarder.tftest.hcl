variables {
  install_policy_json   = <<-EOF
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "cloudformation:*",
          "ec2:DescribeNetworkInterfaces",
          "events:DeleteRule",
          "events:DescribeRule",
          "events:PutRule",
          "events:PutTargets",
          "events:RemoveTargets",
          "iam:AttachRolePolicy",
          "iam:CreateRole",
          "iam:DeleteRole",
          "iam:DeleteRolePolicy",
          "iam:DetachRolePolicy",
          "iam:GetRole",
          "iam:GetRolePolicy",
          "iam:ListAttachedRolePolicies",
          "iam:ListRolePolicies",
          "iam:PassRole",
          "iam:PutRolePolicy",
          "iam:TagRole",
          "iam:UpdateRole",
          "kms:CreateGrant",
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:Encrypt",
          "kms:ListGrants",
          "kms:RevokeGrant",
          "lambda:CreateEventSourceMapping",
          "lambda:CreateFunction",
          "lambda:DeleteEventSourceMapping",
          "lambda:DeleteFunction",
          "lambda:GetEventSourceMapping",
          "lambda:GetFunction",
          "lambda:GetFunctionCodeSigningConfig",
          "lambda:GetRuntimeManagementConfig",
          "lambda:ListEventSourceMappings",
          "lambda:ListTags",
          "lambda:TagResource",
          "lambda:UntagResource",
          "lambda:UpdateEventSourceMapping",
          "lambda:UpdateFunctionCode",
          "lambda:UpdateFunctionConfiguration",
          "logs:CreateLogGroup",
          "logs:DeleteLogGroup",
          "logs:DescribeLogGroups",
          "logs:ListTagsForResource",
          "logs:PutRetentionPolicy",
          "logs:TagResource",
          "s3:GetObject",
          "sqs:CreateQueue",
          "sqs:DeleteQueue",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
          "sqs:PurgeQueue",
          "sqs:SetQueueAttributes"
        ],
        "Resource": "*"
      }
    ]
  }
EOF
  override_match        = "example"
  override_content_type = "application/x-csv;delimiter=space"
}

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
      DataAccessPointArn   = run.target_bucket.access_point.arn
      DestinationUri       = "s3://${run.target_bucket.access_point.alias}"
      SourceBucketNames    = "${join(",", [for k, v in run.sources.buckets : v.id])}"
      SourceTopicArns      = "arn:aws:sns:${run.setup.region}:${run.setup.account_id}:*"
      ContentTypeOverrides = "${var.override_match}=${var.override_content_type}"
      NameOverride         = run.setup.id
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
    queue_arn = run.install_forwarder.stack.outputs["Queue"]
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
      SOURCE      = run.sources.buckets["sqs"].id
      DESTINATION = run.target_bucket.id
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to copy object using SQS"
  }
}

run "check_eventbridge" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.sources.buckets["eventbridge"].id
      DESTINATION = run.target_bucket.id
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
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.sources.buckets["sns"].id
      DESTINATION = run.target_bucket.id
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
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_object_diff"
    env_vars = {
      SOURCE      = run.sources.buckets["sqs"].id
      DESTINATION = run.target_bucket.id
      # this prefix will match the content type override, so we expect the destination object
      # to have our test content type
      OBJECT_PREFIX = var.override_match
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
