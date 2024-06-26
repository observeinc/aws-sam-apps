terraform {
  required_version = ">= 1.1.0"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6"
    }

    aws = {
      source  = "hashicorp/aws"
      version = "~> 5"
    }
  }
}
