![AEM Compose Logo](docs/logo-with-text.png)
[![WTT Logo](docs/wtt-logo.png)](https://www.wundermanthompson.com/service/technology)

[![Apache License, Version 2.0, January 2004](docs/apache-license-badge.svg)](http://www.apache.org/licenses/)

# AEM Compose - Terraform Provider

Allows to manage and provision Adobe Experience Manager (AEM) instances declaratively. 

Built on top of [AEM Compose](https://github.com/wttech/aemc).

## Example usage

```hcl
resource "aws_instance" "aem_author" {
  ami = "ami-043e06a423cbdca17" // RHEL 8
  instance_type = "m5.xlarge"
  associate_public_ip_address = false
  // ...
}

resource "aem_instance" "author" {
  depends_on = [aws_instance.aem_author]
  
  compose {
    root_dir = "/mnt/aemc"
    version = "1.4.1"
    lib_dir = "lib" # files copied once over SCP before creating instance 
    config_file = "aem.yml" # https://github.com/wttech/aemc/blob/0ca8bdeb17be0457ce4bea43621d8abe08948431/pkg/project/app_classic/aem/default/etc/aem.yml
    instance_id = "local_author"
  }
  
  client {
    type = "aws_ssm"
    params = {
      instance_id = aws_instance.aem_author.id
    }
    /*
    type = "ssh"
    params = {
      host = aws_instance.aem_author.*.public_ip
      port = 22
      user = "ec2-user"
      private_key = var.ssh_private_key
    }
    */
  }
}

resource "aem_package" "author_sp17" {
  instance_id = aem_instance.aem_author.id
  name = "sp17"
  file = "aem-service-pkg-6.5.17-1.0.zip"
}

resource "aem_package" "author_mysite_all" {
  instance_id = aem_instance.aem_author.id
  name = "mysite-all"
  file = "mysite-all-1.0.0-SNAPSHOT.zip" # reused from lib dir or copied right before deploy (if needed) 
}

resource "aem_osgi_config" "author_enable_crxde" {
  instance_id = aem_instance.aem_author.id
  name = "enable_crxde"
  pid = "org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet"
  props = {
    alias = "/crx/server"
  }
}

resource "aem_repl_agent" "author_publish" {
  instance_id = aem_instance.aem_author.id
  name = "publish"
  location = "author"
  props = {
    enabled = true
    transportUri = "http://${aem_instance.publish.private_ip}/bin/receive?sling:authRequestLogin=1"
    transportUser = "admin"
    transportPassword = "${var.aem_password}"
    userId = "admin"
  }
}

// ... and similar config for publish instance"
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

For example: `sh develop.sh examples/ssh plan`.

## Debugging the Provider

1. Run command `go run . -debug` from IDEA in debug mode and copy the value of `TF_REATTACH_PROVIDERS` from the output.
2. Set up breakpoint in the code.
3. Run `TF_REATTACH_PROVIDERS=<value_copied_above> TF_CLI_CONFIG_FILE=$(pwd)/dev_overrides.tfrc terraform apply` in one of the examples directory.
4. The breakpoint should be hit.
