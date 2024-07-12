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
          "lambda:CreateEventSourceMapping",
          "lambda:CreateFunction",
          "lambda:DeleteEventSourceMapping",
          "lambda:DeleteFunction",
          "lambda:GetEventSourceMapping",
          "lambda:GetFunction",
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
          "sqs:SetQueueAttributes",
          "sqs:TagQueue"
        ],
        "Resource": "*"
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

run "install" {
  variables {
    setup = run.setup
    app   = "logwriter"
    parameters = {
      BucketArn            = run.create_bucket.arn
      LogGroupNamePatterns = "*"
      DiscoveryRate        = "24 hours"
      NameOverride         = run.setup.id
      Verbosity            = 3
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
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_subscriber"
    env_vars = {
      FUNCTION_ARN = run.install.stack.outputs["SubscriberArn"]
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to verify subscriber invocation"
  }
}

run "update" {
  variables {
    setup = run.setup
    app   = "logwriter"
    parameters = {
      BucketArn            = run.create_bucket.arn
      LogGroupNamePatterns = "*"
      DiscoveryRate        = "24 hours"
      NameOverride         = run.setup.id
      Verbosity            = 4
    }
    capabilities = [
      "CAPABILITY_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}
