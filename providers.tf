variable "aws_access_key" {
	type = string
}

variable "aws_secret_key" {
	type = string
}

terraform {
  required_providers {

    archive = {
      source = "hashicorp/archive"
      version = "2.2.0"
    }

    aws = {
      source = "hashicorp/aws"
      version = "3.70.0"
    }
  }
}

provider "archive" {}

provider "aws" {
  region = "eu-west-2"
  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
}
