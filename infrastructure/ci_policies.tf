data "aws_caller_identity" "current" {}

locals {
  release_bucket_actions = [
    "s3:CreateBucket",
    "s3:GetBucketAcl",
    "s3:GetBucketLocation",
    "s3:GetBucketPolicy",
    "s3:ListBucket",
    "s3:PutBucketAcl",
    "s3:PutBucketPolicy",
  ]

  release_object_actions = [
    "s3:AbortMultipartUpload",
    "s3:CopyObject",
    "s3:DeleteObject",
    "s3:ListMultipartUploadParts",
    "s3:PutObject",
    "s3:PutObjectAcl",
  ]

  cloudformation_actions = [
    "cloudformation:CancelUpdateStack",
    "cloudformation:ContinueUpdateRollback",
    "cloudformation:CreateChangeSet",
    "cloudformation:CreateStack",
    "cloudformation:DeleteChangeSet",
    "cloudformation:DeleteStack",
    "cloudformation:DescribeChangeSet",
    "cloudformation:DescribeStackEvents",
    "cloudformation:DescribeStackResources",
    "cloudformation:DescribeStacks",
    "cloudformation:ExecuteChangeSet",
    "cloudformation:GetTemplate",
    "cloudformation:ListStackResources",
    "cloudformation:ListStacks",
    "cloudformation:UpdateStack",
    "cloudformation:ValidateTemplate",
  ]

  iam_role_actions = [
    "iam:AttachRolePolicy",
    "iam:CreateRole",
    "iam:DeleteRole",
    "iam:DeleteRolePolicy",
    "iam:DetachRolePolicy",
    "iam:GetRole",
    "iam:GetRolePolicy",
    "iam:ListAttachedRolePolicies",
    "iam:ListRolePolicies",
    "iam:PutRolePolicy",
    "iam:TagRole",
    "iam:UpdateRole",
  ]

  iam_passrole_services = [
    "cloudformation.amazonaws.com",
    "config.amazonaws.com",
    "events.amazonaws.com",
    "firehose.amazonaws.com",
    "lambda.amazonaws.com",
    "logs.amazonaws.com",
    "scheduler.amazonaws.com",
  ]

  integration_s3_actions = [
    "s3:CreateAccessPoint",
    "s3:CreateBucket",
    "s3:DeleteAccessPoint",
    "s3:DeleteBucket",
    "s3:DeleteObject",
    "s3:GetAccessPoint",
    "s3:GetAccessPointPolicy",
    "s3:GetBucketNotification",
    "s3:GetBucketTagging",
    "s3:GetLifecycleConfiguration",
    "s3:GetObject",
    "s3:ListAllMyBuckets",
    "s3:ListBucket",
    "s3:PutAccessPointPolicy",
    "s3:PutBucketNotification",
    "s3:PutBucketTagging",
    "s3:PutLifecycleConfiguration",
    "s3:PutObject",
  ]

  stack_resource_actions = [
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
    "kms:CreateAlias",
    "kms:CreateGrant",
    "kms:CreateKey",
    "kms:Decrypt",
    "kms:DeleteAlias",
    "kms:DescribeKey",
    "kms:Encrypt",
    "kms:GetKeyPolicy",
    "kms:ListGrants",
    "kms:PutKeyPolicy",
    "kms:RevokeGrant",
    "kms:ScheduleKeyDeletion",
    "kms:TagResource",
    "lambda:AddPermission",
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
    "lambda:ListFunctions",
    "lambda:ListTags",
    "lambda:RemovePermission",
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
    "logs:DescribeLogStreams",
    "logs:DescribeSubscriptionFilters",
    "logs:ListTagsForResource",
    "logs:PutRetentionPolicy",
    "logs:PutSubscriptionFilter",
    "logs:TagResource",
    "logs:UntagResource",
    "organizations:DescribeAccount",
    "organizations:DescribeOrganization",
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
    "sqs:ListQueueTags",
    "sqs:ListQueues",
    "sqs:PurgeQueue",
    "sqs:SetQueueAttributes",
    "sqs:TagQueue",
    "sqs:UntagQueue",
  ]

  config_reset_actions = [
    "config:DeleteConfigurationRecorder",
    "config:DeleteDeliveryChannel",
    "config:DescribeConfigurationRecorderStatus",
    "config:DescribeConfigurationRecorders",
    "config:DescribeDeliveryChannelStatus",
    "config:DescribeDeliveryChannels",
    "config:StopConfigurationRecorder",
  ]
}

data "aws_iam_policy_document" "ci_release_upload" {
  statement {
    sid       = "ReleaseBuckets"
    effect    = "Allow"
    actions   = local.release_bucket_actions
    resources = ["arn:aws:s3:::observeinc-*"]
  }

  statement {
    sid       = "ReleaseObjects"
    effect    = "Allow"
    actions   = local.release_object_actions
    resources = ["arn:aws:s3:::observeinc-*/aws-sam-apps/*"]
  }

  statement {
    sid       = "Identity"
    effect    = "Allow"
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "ci_integration_tests" {
  statement {
    sid       = "CloudFormationStacks"
    effect    = "Allow"
    actions   = local.cloudformation_actions
    resources = ["*"]
  }

  statement {
    sid     = "SamTransformChangeSets"
    effect  = "Allow"
    actions = ["cloudformation:CreateChangeSet"]
    resources = [
      "arn:aws:cloudformation:*:aws:transform/Serverless-2016-10-31",
      "arn:aws:cloudformation:*:aws:transform/Include",
      "arn:aws:cloudformation:*:aws:transform/LanguageExtensions",
      "arn:aws:cloudformation:*:*:stack/*/*",
    ]
  }

  statement {
    sid       = "ManageInstallRoles"
    effect    = "Allow"
    actions   = local.iam_role_actions
    resources = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/*"]
  }

  statement {
    sid       = "PassRoleToServices"
    effect    = "Allow"
    actions   = ["iam:PassRole"]
    resources = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/*"]
    condition {
      test     = "StringEquals"
      variable = "iam:PassedToService"
      values   = local.iam_passrole_services
    }
  }

  statement {
    sid       = "IntegrationTestBuckets"
    effect    = "Allow"
    actions   = local.integration_s3_actions
    resources = ["*"]
  }

  statement {
    sid       = "StackResources"
    effect    = "Allow"
    actions   = local.stack_resource_actions
    resources = ["*"]
  }

  statement {
    sid       = "ConfigResetScript"
    effect    = "Allow"
    actions   = local.config_reset_actions
    resources = ["*"]
  }

  statement {
    sid    = "CleanupTaggedResources"
    effect = "Allow"
    actions = [
      "cloudformation:DeleteStack",
      "lambda:DeleteFunction",
      "sqs:DeleteQueue",
    ]
    resources = ["*"]
    condition {
      test     = "Null"
      variable = "aws:ResourceTag/github_run_id"
      values   = ["false"]
    }
  }
}

resource "aws_iam_policy" "ci_release_upload" {
  name        = "${local.repository}-gha-ci-release-upload"
  description = "Upload SAM release artifacts to observeinc-* S3 buckets"
  policy      = data.aws_iam_policy_document.ci_release_upload.json
}

resource "aws_iam_policy" "ci_integration_tests" {
  name        = "${local.repository}-gha-ci-integration-tests"
  description = "Run aws-sam-apps integration tests, CloudFormation installs, and cleanup"
  policy      = data.aws_iam_policy_document.ci_integration_tests.json
}
