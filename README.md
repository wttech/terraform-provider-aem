![AEM Compose Logo](docs/logo-with-text.png)
[![WTT Logo](docs/wtt-logo.png)](https://www.wundermanthompson.com/service/technology)

[![Apache License, Version 2.0, January 2004](docs/apache-license-badge.svg)](http://www.apache.org/licenses/)

# AEM Compose - Terraform Provider

Allows to manage and provision Adobe Experience Manager (AEM) instances declaratively. 

Built on top of [AEM Compose](https://github.com/wttech/aemc).

## Examples

### Bare metal example

Assumes that AEM files will be copied from local `lib` directory.

```hcl
resource "aem_instance" "single" {
  files = {
    "lib" = "${local.aem_single_compose_dir}/aem/home/lib"
  }

  system {
    data_dir = local.aem_single_compose_dir
  }
  
  client {
    type = "ssh"
    settings = {
      host   = "x.x.x.x" // IP address of the machine
      user   = "some-user-with-sudo"
    }
    credentials = {
      private_key = file(local.ssh_private_key)
    }
  }

  compose {}
}

locals {
  aem_single_data_dir    = "/data"
  aem_single_compose_dir = "${local.aem_single_data_dir}/aemc"
}
```

### AWS example

Assumes that AEM files are uploaded to S3 bucket `aemc` and are available under `s3://aemc/instance/classic/`.

```hcl
resource "aem_instance" "single" {
  depends_on = [aws_instance.aem_single, aws_volume_attachment.aem_single_data]

  client {
    type = "ssh"
    settings = {
      host   = aws_instance.aem_single.public_ip
      user   = local.ssh_user
    }
    credentials = {
      private_key = file(local.ssh_private_key)
    }
  }

  system {
    data_dir         = local.aem_single_compose_dir
    bootstrap_script = <<SHELL
      #!/bin/sh
      (
        echo "Mounting EBS volume into data directory"
        sudo mkfs -t ext4 ${local.aem_single_data_device} && \
        sudo mkdir -p ${local.aem_single_data_dir} && \
        sudo mount ${local.aem_single_data_device} ${local.aem_single_data_dir} && \
        sudo chown -R ${local.ssh_user} ${local.aem_single_data_dir} && \
        echo '${local.aem_single_data_device} ${local.aem_single_data_dir} ext4 defaults 0 0' | sudo tee -a /etc/fstab
      ) && (
        echo "Copying AEM library files"
        sudo yum install -y unzip && \
        curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
        unzip -q awscliv2.zip && \
        sudo ./aws/install --update && \
        mkdir -p "${local.aem_single_compose_dir}/aem/home/lib" && \
        aws s3 cp --recursive --no-progress "s3://aemc/instance/classic/" "${local.aem_single_compose_dir}/aem/home/lib"
      )
    SHELL
  }
  
  compose {}
}

locals {
  // https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/device_naming.html#device-name-limits
  aem_single_data_device = "/dev/nvme1n1"
  aem_single_data_dir    = "/data"
  aem_single_compose_dir = "${local.aem_single_data_dir}/aemc"
}

output "aem_instances" {
  value = aem_instance.single.instances
}

```

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

- `sh develop.sh examples/ssh plan`
- `sh develop.sh examples/ssh apply -auto-approve`
- `sh develop.sh examples/ssh destroy -auto-approve`

## Debugging the Provider

1. Run command `go run . -debug` from IDEA in debug mode and copy the value of `TF_REATTACH_PROVIDERS` from the output.
2. Set up breakpoint in the code.
3. Run `TF_REATTACH_PROVIDERS=<value_copied_above> TF_CLI_CONFIG_FILE=$(pwd)/../../dev_overrides.tfrc terraform apply` in one of the examples directory.
4. The breakpoint should be hit.
