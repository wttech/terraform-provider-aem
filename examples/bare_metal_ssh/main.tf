terraform {
  required_providers {
    aem = {
      source  = "registry.terraform.io/wttech/aem"
      version = "< 2.0.0"
    }
  }
}

provider "aem" {}
