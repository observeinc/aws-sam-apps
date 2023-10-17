run "setup" {
  module {
    source = "./tests/setup"
  }
}

run "install_forwarder" {
  variables {
    name = run.setup.id
    app = "forwarder"
    parameters = {
      DataAccessPointArn = run.setup.source.arn
      DestinationUri     = "s3://${run.setup.source.alias}"
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}
