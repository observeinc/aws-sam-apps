# Verifies the PermissionsBoundary parameter added by the SAM templates is
# actually applied to every IAM::Role created by the deployed stack.
#
# The boundary ARN used here is the AWS-managed AdministratorAccess policy.
# The test never invokes the resulting roles; it only checks that the
# PermissionsBoundary attribute on each created role matches. Any real
# customer-supplied boundary would exercise the same plumbing.

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

  boundary_arn = "arn:aws:iam::aws:policy/AdministratorAccess"
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

run "install_with_boundary" {
  variables {
    setup = run.setup
    app   = "forwarder"
    parameters = {
      DestinationUri      = "s3://${run.target_bucket.id}/"
      SourceObjectKeys    = "*"
      NameOverride        = run.setup.id
      PermissionsBoundary = var.boundary_arn
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "verify_boundary_applied" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "./scripts/check_role_boundaries"
    env_vars = {
      STACK_NAME        = run.setup.stack_name
      EXPECTED_BOUNDARY = var.boundary_arn
      AWS_REGION        = run.setup.region
    }
  }

  assert {
    condition     = output.exitcode == 0
    error_message = "One or more IAM roles in the stack do not carry the expected PermissionsBoundary"
  }
}
