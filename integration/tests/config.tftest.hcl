run "setup" {
  module {
    source = "./modules/setup/run"
  }
}

run "install_config" {
  variables {
    name = run.setup.id
    app  = "config"
    parameters = {
      BucketName = run.setup.access_point.bucket
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "check" {
  module {
    source = "./modules/exec"
  }

  # TODO: this check is a bit weak, since it only verifies
  # that something was written. AWS Config immediately writes a
  # ConfigWritabilityCheckFile to the destination bucket, so we at least verify
  # it has write privileges.
  # A better check would be to verify snapshot data is delivered.
  variables {
    command = "./scripts/check_bucket_not_empty"
    env_vars = {
      SOURCE = run.setup.access_point.bucket
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Bucket is empty"
  }
}

run "install_include" {
  variables {
    name = run.setup.id
    app  = "config"
    parameters = {
      BucketName    = run.setup.access_point.bucket
      IncludeResourceTypes = "AWS::Redshift::ClusterSnapshot,AWS::RDS::DBClusterSnapshot,AWS::CloudFront::StreamingDistribution"
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}

run "install_exclude" {
  variables {
    name = run.setup.id
    app  = "config"
    parameters = {
      BucketName    = run.setup.access_point.bucket
      ExcludeResourceTypes = "AWS::Redshift::ClusterSnapshot,AWS::RDS::DBClusterSnapshot,AWS::CloudFront::StreamingDistribution"
    }
    capabilities = [
      "CAPABILITY_NAMED_IAM",
      "CAPABILITY_AUTO_EXPAND",
    ]
  }
}