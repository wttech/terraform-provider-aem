#!/bin/bash

TF_RC_FILE="/Users/$(whoami)/.terraformrc"

echo "Removing Terraform CLI configuration file: $TF_RC_FILE"
rm  "$TF_RC_FILE"
