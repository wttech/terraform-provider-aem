terraform {
  required_providers {
    aem = {
      source = "registry.terraform.io/wttech/aem"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.14.0"
    }
  }
}

provider "aem" {}

#provider "aws" {
#  region = "eu-central-1"
#}

resource "aws_instance" "aem_author" {
  ami           = "ami-043e06a423cbdca17"
  instance_type = "t2.micro"
}
