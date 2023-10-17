run "setup" {
  module {
    source = "./tests/setup"
  }
}

run "check" {
  module {
    source = "./tests/check"
  }

  variables {
    program = ["./scripts/test.sh", run.setup.id ]
  }

  assert {
    condition     = output.result.error == ""
    error_message = "Failed to test thing"
  }
}
