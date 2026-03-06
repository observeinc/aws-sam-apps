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
          "iam:UpdateRole"
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

run "install_config" {
  variables {
    setup = run.setup
    app   = "config"
    parameters = {
      BucketName = run.create_bucket.id
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "check" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  # TODO: this check is a bit weak, since it only verifies
  # that something was written. AWS Config immediately writes a
  # ConfigWritabilityCheckFile to the destination bucket, so we at least verify
  # it has write privileges.
  # A better check would be to verify snapshot data is delivered.
  variables {
    command = "./scripts/check_bucket_not_empty"
    env_vars = {
      SOURCE = run.create_bucket.id
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Bucket is empty"
  }
}

run "install_include" {
  variables {
    setup = run.setup
    app   = "config"
    parameters = {
      BucketName           = run.create_bucket.id
      IncludeResourceTypes = "AWS::Redshift::ClusterSnapshot,AWS::RDS::DBClusterSnapshot"
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "install_exclude" {
  variables {
    setup = run.setup
    app   = "config"
    parameters = {
      BucketName           = run.create_bucket.id
      ExcludeResourceTypes = "AWS::Redshift::ClusterSnapshot,AWS::RDS::DBClusterSnapshot"
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}
