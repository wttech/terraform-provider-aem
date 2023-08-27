#!/bin/bash

TF_DIR=$1
if [ -z "$TF_DIR" ]
then
  echo "Usage: $0 <terraform_dir> <terraform_args>"
  exit 1
fi

echo "Building and installing Terraform AEM provider"
go install .

TF_CLI_CONFIG_FILE="$(pwd)/dev_overrides.tfrc"

echo "Executing Terraform command at dir: $TF_DIR"
(export TF_CLI_CONFIG_FILE && cd "$TF_DIR" && terraform "${@:2}")
