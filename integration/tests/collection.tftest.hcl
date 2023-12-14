variables {
  install_policy_json = <<-EOF
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "cloudformation:CreateChangeSet",
          "cloudformation:CreateStack",
          "cloudformation:DeleteStack",
          "cloudformation:DescribeStacks",
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
      }
    ]
  }
EOF
}

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
