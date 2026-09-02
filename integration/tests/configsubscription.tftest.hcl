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
          "events:DeleteRule",
          "events:DescribeRule",
          "events:PutRule",
          "events:PutTargets",
          "events:RemoveTargets"
        ],
        "Resource": "*"
      }
    ]
  }
EOF
}

run "setup" {
  module {
    source = "./modules/setup"
  }
}

run "target" {
  module {
    source  = "observeinc/collection/aws//modules/testing/sqs_queue"
    version = "2.14.0"
  }

  variables {
    setup = run.setup
  }
}


run "install" {
  variables {
    setup = run.setup
    app   = "configsubscription"
    parameters = {
      TargetArn = run.target.queue.arn
    }
    capabilities = [
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}
