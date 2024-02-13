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
          "iam:TagRole",
          "iam:UpdateRole",
          "kms:CreateGrant",
          "kms:Decrypt",
          "kms:DescribeKey",
          "kms:Encrypt",
          "lambda:CreateEventSourceMapping",
          "lambda:CreateFunction",
          "lambda:DeleteEventSourceMapping",
          "lambda:DeleteFunction",
          "lambda:GetEventSourceMapping",
          "lambda:GetFunction",
          "lambda:ListEventSourceMappings",
          "lambda:TagResource",
          "lambda:UntagResource",
          "lambda:UpdateEventSourceMapping",
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:DeleteLogGroup",
          "logs:DeleteLogStream",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams",
          "logs:ListTagsForResource",
          "logs:PutRetentionPolicy",
          "logs:TagResource",
          "s3:GetObject",
          "scheduler:CreateSchedule",
          "scheduler:DeleteSchedule",
          "scheduler:GetSchedule",
          "scheduler:UpdateSchedule",
          "sqs:CreateQueue",
          "sqs:DeleteQueue",
          "sqs:GetQueueAttributes",
          "sqs:SetQueueAttributes"
        ],
        "Resource": "*"
      }
    ]
  }
EOF
}

run "setup" {
  module {
    source  = "observeinc/collection/aws//modules/testing/run"
    version = "2.6.0"
  }
}

run "install" {
  variables {
    setup = run.setup
    app   = "logwriter"
    parameters = {
      BucketARN            = "arn:aws:s3:::${run.setup.access_point.bucket}"
      LogGroupNamePatterns = "*"
      DiscoveryRate        = "24 hours"
      NameOverride         = run.setup.id
    }
    capabilities = [
      "CAPABILITY_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "check_eventbridge_invoked" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.6.0"
  }

  variables {
    command = "./scripts/check_subscriber"
    env_vars = {
      FUNCTION_ARN = run.install.stack.outputs["Function"]
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to verify subscriber invocation"
  }
}
