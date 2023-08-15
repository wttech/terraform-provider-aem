![AEM Compose Logo](https://github.com/wttech/terraform-provider-aem/raw/main/images/logo-with-text.png)
[![WTT Logo](https://github.com/wttech/terraform-provider-aem/raw/main/images/wtt-logo.png)](https://www.wundermanthompson.com/service/technology)

[![Last Release Version](https://img.shields.io/github/v/release/wttech/aemc?color=lightblue&label=Last%20Release)](https://github.com/wttech/terraform-provider-aem/tags)
[![Ansible Galaxy](https://img.shields.io/ansible/collection/2218?label=Ansible%20Galaxy)](https://galaxy.ansible.com/wttech/aem)
[![Apache License, Version 2.0, January 2004](https://github.com/wttech/terraform-provider-aem/raw/main/images/apache-license-badge.svg)](http://www.apache.org/licenses/)

# AEM Compose - Terraform Provider

Allows to manage and provision Adobe Experience Manager (AEM) instances declaratively. 

Built on top of [AEM Compose](https://github.com/wttech/aemc).

## Example usage

```hcl
resource "aws_instance" "aem_author" {
  // ...
}

resource "aem_instance" "author" {

  config {
    port = 4502
    password = var.aem_password
    run_modes = ["nosamplecontent", "int"]
    jvm_opts = [
      "-server",
      "-Djava.awt.headless=true",
      "-Djava.io.tmpdir=aem/home/tmp",
      "-agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=0.0.0.0:14502",
      "-Duser.language=en",
      "-Duser.country=US",
      "-Duser.timezone=UTC",
    ]
    start_opts: []
    secret_vars = [
      "ACME_SECRET=value",
    ]
    env_vars = [
      "ACME_VAR=value",
    ]
    sling_props: []
  }

  connection {
    type = "ssh"
    params = {
      user: "ec2-user"
      private_key: var.ssh_private_key
    }
    
    /*
    type = "aws_ssm"
    params = {
      instance_id: aws_instance.aem_author.id
    }
    */
  }
}

resource aem_package "author_mysite_all" {
  name = "mysite-all"
  instance_id = aem_instance.aem_author.id
  file = "mysite-all-1.0.0-SNAPSHOT.zip"
}

resource "aem_package" "author_mysite_content" {
  name = "mysite-sample-content"
  instance_id = aem_instance.aem_author.id
  file = "mysite-sample-content-1.0.0-SNAPSHOT.zip"
}

resource "aem_osgi_config" "author_enable_crxde" {
  name = "enable_crxde"
  instance_id = aem_instance.aem_author.id
  pid = "org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet"
  props = {
    "alias": "/crx/server"
  }
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
