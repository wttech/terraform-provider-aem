![AEM Compose Logo](docs/logo-with-text.png)
[![WTT Logo](docs/wtt-logo.png)](https://www.wundermanthompson.com/service/technology)

[![Apache License, Version 2.0, January 2004](docs/apache-license-badge.svg)](http://www.apache.org/licenses/)

# AEM Compose - Terraform Provider

This provider allows development teams to easily set up [Adobe Experience Manager (AEM)](https://business.adobe.com/products/experience-manager/adobe-experience-manager.html) instances on virtual machines in the cloud (AWS, Azure, GCP, etc.) or bare metal machines.
It's based on the [AEM Compose](https://github.com/wttech/aemc) tool and aims to simplify the process of creating AEM environments without requiring deep DevOps knowledge.

Published in [Terraform Registry](https://registry.terraform.io/providers/wttech/aem/latest/docs).

## Purpose

The main purpose of this provider is to enable users to:

- Set up as many AEM environments as needed with minimal effort
- Eliminate the need for deep DevOps knowledge
- Allow for seamless integration with popular cloud platforms such as AWS and Azure
- Provide a simple and efficient way to manage AEM instances

## Features

- Easy configuration and management of AEM instances
- Support for multiple cloud platforms and bare metal machines
- Seamless integration with Terraform for infrastructure provisioning
- Based on the powerful [AEM Compose](https://github.com/wttech/aemc) tool


## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Fill this in for each provider

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

## Testing the Provider using examples

Run command: `sh develop.sh <example_path> <tf_args>`.

For example: 

- `sh develop.sh examples/aws_ssh plan`
- `sh develop.sh examples/aws_ssh apply -auto-approve`
- `sh develop.sh examples/aws_ssh destroy -auto-approve`

## Debugging the Provider

1. Run command `go run . -debug` from IDEA in debug mode and copy the value of `TF_REATTACH_PROVIDERS` from the output.
2. Set up breakpoint in the code.
3. Run `TF_REATTACH_PROVIDERS=<value_copied_above> TF_CLI_CONFIG_FILE=$(pwd)/../../dev_overrides.tfrc terraform apply` in one of the examples directory.
4. The breakpoint should be hit.
