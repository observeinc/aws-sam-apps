variables {
  install_policy_json = <<-EOF
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "cloudformation:*",
          "iam:CreateRole",
          "iam:DeleteRole",
          "iam:UpdateRole",
          "iam:DeleteRolePolicy",
          "iam:GetRole",
          "iam:GetRolePolicy",
          "iam:ListAttachedRolePolicies",
          "iam:ListRolePolicies",
          "iam:PutRolePolicy",
          "iam:PassRole",
          "iam:AttachRolePolicy",
          "iam:DetachRolePolicy",
          "logs:DescribeLogGroups",
          "logs:ListTagsForResource",
          "firehose:CreateDeliveryStream",
          "firehose:DeleteDeliveryStream",
          "firehose:DescribeDeliveryStream",
          "firehose:ListTagsForDeliveryStream",
          "firehose:UpdateDestination",
          "logs:CreateLogGroup",
          "logs:DeleteLogGroup",
          "logs:PutRetentionPolicy",
          "logs:CreateLogStream",
          "logs:DeleteLogStream",
          "logs:DescribeLogStreams"
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

run "install" {
  variables {
    name = run.setup.id
    app  = "firehose"
    parameters = {
      BucketARN = "arn:aws:s3:::${run.setup.access_point.bucket}"
    }
    capabilities = [
      "CAPABILITY_IAM",
    ]
  }
}

run "check_firehose" {
  module {
    source = "./modules/exec"
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
    name = run.setup.id
    app  = "firehose"
    parameters = {
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
    source = "./modules/exec"
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
