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

## Overview

Provides an AEM instance resource to set up one or more AEM instances on virtual machines in the cloud or bare metal machines.
Below configuration is a generic example of how to use the provider:

```hcl
resource "aem_instance" "single" {
  depends_on = [] // for example: [aws_instance.aem_single, aws_volume_attachment.aem_single_data]

  // see available connection types: https://github.com/wttech/terraform-provider-aem/blob/main/internal/client/client_manager.go
  client { 
    type = "<type>"  // 'aws-ssm' or 'ssh'
    settings = {
      // type-specific values goes here
    }
    credentials = {
      // type-specific values goes here
    }
  }

  system {
    bootstrap = {
      inline = [
        // commands to execute only once on the machine (not idempotent)
      ]
    }
  }

  compose {
    create = {
      inline = [
        // commands to execute before launching AEM instances (idempotent)
        // for downloading AEM files, etc.
      ]
    }
    configure = {
      inline = [
        // commands to execute after launching AEM instances (idempotent)
        // for provisioning AEM instances: setting replication agents, installing packages, etc.
      ]
    }
  }
}

output "aem_instances" {
  value = aem_instance.single.instances
}
```

## Quickstart

The easiest way to get started is to review, copy and adapt provided examples:

1. [AWS EC2 instance with private IP](https://github.com/wttech/terraform-provider-aem/tree/main/examples/aws_ssm)
2. [AWS EC2 instance with public IP](https://github.com/wttech/terraform-provider-aem/tree/main/examples/aws_ssh)
3. [Bare metal machine](https://github.com/wttech/terraform-provider-aem/tree/main/examples/bare_metal_ssh)

- - -

## Development

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

### Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command:

```shell
go install
```

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

### Troubleshooting the provider

Before running any Terraform command simply set the environment variable `TF_LOG=INFO` (or ultimately `TF_LOG=DEBUG`) to see detailed logs about progress of the setting up the AEM instances.

### Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

### Testing the Provider using examples

Run command: `sh develop.sh <example_path> <tf_args>`.

For example: 

- `sh develop.sh examples/aws_ssh plan`
- `sh develop.sh examples/aws_ssh apply -auto-approve`
- `sh develop.sh examples/aws_ssh destroy -auto-approve`

- `sh develop.sh examples/aws_ssm plan`
- `sh develop.sh examples/aws_ssm apply -auto-approve`
- `sh develop.sh examples/aws_ssm destroy -auto-approve`

### Debugging the Provider

1. Run command `go run . -debug` from IDEA in debug mode and copy the value of `TF_REATTACH_PROVIDERS` from the output.
2. Set up breakpoint in the code.
3. Run `TF_REATTACH_PROVIDERS=<value_copied_above> TF_CLI_CONFIG_FILE=$(pwd)/../../dev_overrides.tfrc terraform apply` in one of the examples directory.
4. The breakpoint should be hit.
