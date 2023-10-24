run "check_file_not_copied" {
  module {
    source = "./tests/check"
  }

  variables {
    command = "./scripts/check_object_copy"
    env_vars = {
      SOURCE      = var.source_bucket
      DESTINATION = var.destination_arn
    }
  }

  assert {
    condition     = output.error == "failed to read file from destination"
    error_message = "Unexpected error"
  }
}

run "subscribe_bucket_notifications_to_eventbridge" {
  module {
    source = "./tests/bucket_eventbridge"
  }

  variables {
    bucket      = var.source_bucket
  }
}

# run "check_copy_succeeds" {
#   module {
#     source = "./tests/check"
#   }

#   variables {
#     command  = "./scripts/check_object_copy"
#     env_vars = {
#       SOURCE      = var.source_bucket
#       DESTINATION = var.destination_arn
#     }
#   }

#   assert {
#     condition     = output.error == ""
#     error_message = "Failed to copy object"
#   }
# }
