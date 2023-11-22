run "setup" {
  module {
    source = "./modules/setup/run"
  }
}

run "install_with_discovery" {
  variables {
    name = run.setup.id
    app  = "subscriber"
    parameters = {
      LogGroupNamePatterns = "*"
      DiscoveryRate        = "1 hour"
    }
    capabilities = [
      "CAPABILITY_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "check_eventbridge_invoked" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_subscriber"
    env_vars = {
      FUNCTION_ARN = run.install_with_discovery.stack.outputs["Function"]
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to verify subscriber invocation"
  }
}

// verify all our conditional logic is good when reconfiguring options
run "install_without_discovery" {
  variables {
    name = run.setup.id
    app  = "subscriber"
    parameters = {}
    capabilities = [
      "CAPABILITY_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}
