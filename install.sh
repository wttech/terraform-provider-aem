#!/bin/bash

GO_BIN_DIR="/Users/$(whoami)/go/bin"
TF_RC_FILE="$(pwd)/dev_overrides.tfrc"

# Download all providers and pretend that they are developed locally
# See: https://github.com/hashicorp/terraform/issues/27459#issuecomment-1382126220

# AWS provider
if [ ! -f "$GO_BIN_DIR/terraform-provider-aws" ]
then
  echo "Setting up Terraform AWS provider as dev-override: $GO_BIN_DIR/terraform-provider-aws"
  wget https://releases.hashicorp.com/terraform-provider-aws/5.14.0/terraform-provider-aws_5.14.0_darwin_arm64.zip -c -O /tmp/terraform-provider-aws.zip
  unzip -o /tmp/terraform-provider-aws.zip -d "$GO_BIN_DIR"
  cp /tmp/terraform-provider-aws_v5.14.0_x5 "$GO_BIN_DIR/terraform-provider-aws"
fi

# TLS provider
if [ ! -f "$GO_BIN_DIR/terraform-provider-tls" ]
then
  echo "Setting up Terraform TLS provider as dev-override: $GO_BIN_DIR/terraform-provider-tls"
  wget https://releases.hashicorp.com/terraform-provider-tls/4.0.4/terraform-provider-tls_4.0.4_darwin_arm64.zip -c -O /tmp/terraform-provider-tls.zip
  unzip -o /tmp/terraform-provider-tls.zip -d "$GO_BIN_DIR"
  cp /tmp/terraform-provider-tls_v4.0.4_x5 "$GO_BIN_DIR/terraform-provider-tls"
fi

echo "Setting up dev-overrides in custom Terraform CLI configuration file: $TF_RC_FILE"
cat <<EOT > "$TF_RC_FILE"
provider_installation {

  dev_overrides {
      "registry.terraform.io/wttech/aem" = "$GO_BIN_DIR"
      "registry.terraform.io/hashicorp/aws" = "$GO_BIN_DIR"
      "registry.terraform.io/hashicorp/tls" = "$GO_BIN_DIR"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
EOT

echo "Building and installing Terraform AEM provider"
go install .

echo ""

echo "To use the Terraform AEM provider, add the following to ZSH RC:"
echo "export TF_CLI_CONFIG_FILE=$TF_RC_FILE >> ~/.zshrc"

echo ""

echo "Alternatively, just run following command in shell right before running Terraform commands like 'plan', 'apply', etc.:"
echo "export TF_CLI_CONFIG_FILE=$TF_RC_FILE"
