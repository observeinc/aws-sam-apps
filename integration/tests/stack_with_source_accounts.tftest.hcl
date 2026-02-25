variables {
  # Test with a fake account ID that would be from another account in the org
  test_source_account_id = "123456789012"

  install_policy_json = <<-EOF
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "cloudformation:CreateStack",
          "cloudformation:DeleteChangeSet",
          "cloudformation:DeleteStack",
          "cloudformation:DescribeStacks",
          "cloudwatch:DeleteMetricStream",
          "cloudwatch:GetMetricStream",
          "cloudwatch:PutMetricStream",
          "cloudwatch:TagResource",
          "config:DeleteConfigurationRecorder",
          "config:DeleteDeliveryChannel",
          "config:DescribeConfigurationRecorderStatus",
          "config:DescribeConfigurationRecorders",
          "config:DescribeDeliveryChannelStatus",
          "config:DescribeDeliveryChannels",
          "config:PutConfigurationRecorder",
          "config:PutDeliveryChannel",
          "config:StartConfigurationRecorder",
          "config:StopConfigurationRecorder",
          "ec2:DescribeNetworkInterfaces",
          "events:DeleteRule",
          "events:DescribeRule",
          "events:PutRule",
          "events:PutTargets",
          "events:RemoveTargets",
          "firehose:CreateDeliveryStream",
          "firehose:DeleteDeliveryStream",
          "firehose:DescribeDeliveryStream",
          "firehose:ListTagsForDeliveryStream",
          "firehose:TagDeliveryStream",
          "firehose:UpdateDestination",
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
          "lambda:InvokeFunction",
          "lambda:ListEventSourceMappings",
          "lambda:ListTags",
          "lambda:TagResource",
          "lambda:UntagResource",
          "lambda:UpdateEventSourceMapping",
          "lambda:UpdateFunctionCode",
          "lambda:UpdateFunctionConfiguration",
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:DeleteLogGroup",
          "logs:DeleteLogStream",
          "logs:DeleteSubscriptionFilter",
          "logs:DescribeLogGroups",
          "logs:DescribeSubscriptionFilters",
          "logs:ListTagsForResource",
          "logs:PutRetentionPolicy",
          "logs:PutSubscriptionFilter",
          "logs:TagResource",
          "logs:UntagResource",
          "s3:CreateBucket",
          "s3:DeleteBucket",
          "s3:GetBucketNotification",
          "s3:GetLifecycleConfiguration",
          "s3:GetObject",
          "s3:ListBucket",
          "s3:PutBucketNotification",
          "s3:PutBucketTagging",
          "s3:PutLifecycleConfiguration",
          "scheduler:CreateSchedule",
          "scheduler:DeleteSchedule",
          "scheduler:GetSchedule",
          "scheduler:UpdateSchedule",
          "sns:CreateTopic",
          "sns:DeleteTopic",
          "sns:GetTopicAttributes",
          "sns:ListTopics",
          "sns:Publish",
          "sns:SetTopicAttributes",
          "sns:Subscribe",
          "sns:TagResource",
          "sns:Unsubscribe",
          "sqs:CreateQueue",
          "sqs:DeleteQueue",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
          "sqs:PurgeQueue",
          "sqs:SetQueueAttributes",
          "sqs:TagQueue",
          "sqs:UntagQueue"
        ],
        "Resource": "*"
      },
      {
        "Effect": "Allow",
        "Action": [
          "cloudformation:CreateChangeSet"
        ],
        "Resource": [
          "arn:aws:cloudformation:*:aws:transform/Serverless-2016-10-31",
          "arn:aws:cloudformation:*:aws:transform/Include",
          "arn:aws:cloudformation:*:aws:transform/LanguageExtensions",
          "arn:aws:cloudformation:*:*:stack/*/*"
        ]
      },
      {
        "Effect": "Allow",
        "Action": [
          "s3:GetObject"
        ],
        "Resource": [
          "arn:aws:s3:::observeinc/cloudwatchmetrics/filters/*"
        ]
      }
    ]
  }
EOF
}

run "setup" {
  module {
    source  = "observeinc/collection/aws//modules/testing/setup"
    version = "2.9.0"
  }
  variables {
    id_length = 51
  }
}

run "create_bucket" {
  module {
    source  = "observeinc/collection/aws//modules/testing/s3_bucket"
    version = "2.9.0"
  }
  variables {
    setup = run.setup
  }
}

# Get the current account ID for testing
run "get_account_id" {
  module {
    source = "./modules/get_org_id"
  }
}

run "reset_config_service" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/reset_config_service"
    env_vars = {
      RESET_CONFIG_SERVICE = "1"
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to reset AWS Config service state"
  }
}

run "install_with_source_accounts" {
  variables {
    setup = run.setup
    app   = "stack"
    parameters = {
      DataAccessPointArn       = run.create_bucket.access_point.arn
      DestinationUri           = "s3://${run.create_bucket.access_point.alias}/"
      ConfigDeliveryBucketName = "example-bucket"
      SourceBucketNames        = "*"
      LogGroupNamePatterns     = "*"
      SourceAccounts           = var.test_source_account_id
      NameOverride             = run.setup.id
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "verify_source_accounts_policy" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_sns_topic_policy"
    env_vars = {
      TOPIC_ARN            = run.install_with_source_accounts.stack.outputs["TopicArn"]
      EXPECTED_ACCOUNT_IDS = var.test_source_account_id
      CURRENT_ACCOUNT_ID   = run.get_account_id.account_id
      CHECK_TYPE           = "source_accounts"
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "SNS topic policy does not contain expected SourceAccounts condition with current account ID"
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
      SOURCE      = run.install_with_source_accounts.stack.outputs["BucketName"]
      DESTINATION = run.create_bucket.id
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "Failed to copy object using SQS"
  }
}
