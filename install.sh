#!/bin/bash

TF_RC_FILE="/Users/$(whoami)/.terraformrc"
GO_BIN_DIR="/Users/$(whoami)/go/bin"

# Download all providers and pretend that they are developed locally
# See: https://github.com/hashicorp/terraform/issues/27459#issuecomment-1382126220

# AWS provider
if [ ! -f "$GO_BIN_DIR/terraform-provider-aws" ]
then
  echo "Setting up Terraform AWS provider as dev-override: $GO_BIN_DIR/terraform-provider-aws"
  wget https://releases.hashicorp.com/terraform-provider-aws/5.14.0/terraform-provider-aws_5.14.0_darwin_arm64.zip -O /tmp/terraform-provider-aws.zip
  unzip /tmp/terraform-provider-aws.zip -d "$GO_BIN_DIR"
  cp /tmp/terraform-provider-aws_v5.14.0_x5 "$GO_BIN_DIR/terraform-provider-aws"
fi

echo "Setting up dev-overrides in Terraform CLI configuration file: $TF_RC_FILE"
cat <<EOT > "$TF_RC_FILE"
provider_installation {

  dev_overrides {
      "registry.terraform.io/wttech/aem" = "/Users/$(whoami)/go/bin"
      "registry.terraform.io/hashicorp/aws" = "/Users/$(whoami)/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
EOT

echo "Building and installing Terraform AEM provider"
go install .
