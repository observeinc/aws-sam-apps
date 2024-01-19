variables {
  install_policy_json = <<-EOF
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "cloudformation:CreateStack",
          "cloudformation:DeleteStack",
          "cloudformation:DescribeStacks",
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
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:DeleteLogGroup",
          "logs:DeleteLogStream",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams",
          "logs:ListTagsForResource",
          "logs:PutRetentionPolicy",
          "logs:TagResource"
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
    app   = "firehose"
    parameters = {
      NameOverride = run.setup.id
      BucketARN    = "arn:aws:s3:::${run.setup.access_point.bucket}"
    }
    capabilities = [
      "CAPABILITY_IAM",
    ]
  }
}

run "check_firehose" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.6.0"
  }

  variables {
    command = "./scripts/check_firehose"
    env_vars = {
      FIREHOSE_ARN = run.install.stack.outputs["Firehose"]
      DESTINATION  = "s3://${run.setup.access_point.bucket}/"
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to write firehose records"
  }
}

run "set_prefix" {
  variables {
    setup = run.setup
    app   = "firehose"
    parameters = {
      NameOverride      = run.setup.id
      BucketARN         = "arn:aws:s3:::${run.setup.access_point.bucket}"
      Prefix            = "${run.setup.id}/"
      WriterRoleService = "logs.amazonaws.com"
    }
    capabilities = [
      "CAPABILITY_IAM",
    ]
  }
}

run "check_firehose_prefix" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.6.0"
  }

  variables {
    command = "./scripts/check_firehose"
    env_vars = {
      FIREHOSE_ARN = run.install.stack.outputs["Firehose"]
      DESTINATION  = "s3://${run.setup.access_point.bucket}/${run.setup.id}/"
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to write firehose records"
  }
}
