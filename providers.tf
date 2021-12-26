variable "aws_access_key" {
	type = string
}

variable "aws_secret_key" {
	type = string
}

variable "cloudflare_email" {
	type = string
}

variable "cloudflare_api_token" {
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

    cloudflare = {
      source = "cloudflare/cloudflare"
      version = "3.6.0"
    }

  }
}

provider "archive" {}

provider "aws" {
  region = "eu-west-2"
  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
}

provider "aws" {
  alias = "us"
  region = "us-east-1"
  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
}

provider "cloudflare" {
  email = var.cloudflare_email
  api_token = var.cloudflare_api_token
}
