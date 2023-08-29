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

locals {
  workspace = "aemc"
  env_type  = "aem-single"
  host      = "aem_single"

  ssh_user        = "ec2-user"
  ssh_private_key = file("ec2-key.cer")

  tags = {
    Workspace = "aemc"
    Env       = "tf-minimal"
    EnvType   = "aem-single"
    Host      = "aem-single"
    Name      = "${local.workspace}_${local.env_type}_${local.host}"
  }
}

resource "aws_instance" "aem_author" {
  ami                         = "ami-043e06a423cbdca17" // RHEL 8
  instance_type               = "m5.xlarge"
  associate_public_ip_address = true

  tags = local.tags
}

output "instance_ip" {
  value = aws_instance.aem_author.public_ip
}

resource "aem_instance" "author" {
  depends_on = [aws_instance.aem_author]

  client {
    type = "ssh"
    settings = {
      host        = aws_instance.aem_author.public_ip
      port        = 22
      user        = "ec2-user"
     // private_key = local.ssh_private_key // TODO hide it in TF console outputs
    }
  }
  compose {
    data_dir    = "/data/aemc"
    version     = "1.4.1"
    lib_dir     = "lib"
    config_file = "aem.yml"
    instance_id = "local_author"
  }
}
