#!/bin/bash

TF_DIR=$1
if [ -z "$TF_DIR" ]
then
  echo "Usage: $0 <terraform_dir> <terraform_args>"
  exit 1
fi

TF_CLI_CONFIG_FILE="$(pwd)/dev_overrides.tfrc"

(export TF_CLI_CONFIG_FILE && cd "$TF_DIR" && terraform "${@:2}")
