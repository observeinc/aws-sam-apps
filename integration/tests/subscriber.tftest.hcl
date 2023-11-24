run "setup" {
  module {
    source = "./modules/setup/run"
  }
}

run "install" {
  variables {
    name = run.setup.id
    app  = "subscriber"
    parameters = {
    }
    capabilities = [
      "CAPABILITY_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "check_invoke" {
  module {
    source = "./modules/exec"
  }

  variables {
    command = "./scripts/check_subscriber"
    env_vars = {
      FUNCTION_ARN = run.install.stack.outputs["Function"]
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Failed to invoke lambda function"
  }
}
