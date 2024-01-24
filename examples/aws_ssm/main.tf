terraform {
  required_providers {
    aem = {
      source  = "registry.terraform.io/wttech/aem"
      version = "< 2.0.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.14.0"
    }
  }
}

provider "aem" {}

locals {
  workspace = "aemc"
  env_type  = "aem-single"
  host      = "aem_single"

  ssm_user = "root"

  tags = {
    Workspace = "aemc"
    Env       = "tf-minimal"
    EnvType   = "aem-single"
    Host      = "aem-single"
    Name      = "${local.workspace}_${local.env_type}_${local.host}"
  }
}
