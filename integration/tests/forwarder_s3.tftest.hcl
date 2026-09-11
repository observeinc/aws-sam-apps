provider "aws" {
  default_tags {
    tags = {
      "managed-by" = "integration-test"
    }
  }
}

variables {
  install_policy_json = <<-EOF
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
          "sqs:SetQueueAttributes",
          "sqs:TagQueue"
        ],
        "Resource": "*"
      }
    ]
  }
EOF
}

# This test verifies our forwarder can write to an S3 bucket directly,
# without being fronted by a DataAccessPoint
run "setup" {
  module {
    source = "./modules/setup"
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
      SourceBucketNames = "${run.sources.buckets["sns"].id},${run.sources.buckets["sqs"].id},${run.sources.buckets["eventbridge"].id},${run.sources.buckets["kms"].id}"
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
      COPY_DELAY    = 10
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
