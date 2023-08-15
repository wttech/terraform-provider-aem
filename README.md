![AEM Compose Logo](https://github.com/wttech/terraform-aem-provider/raw/main/images/logo-with-text.png)
[![WTT Logo](https://github.com/wttech/terraform-aem-provider/raw/main/images/wtt-logo.png)](https://www.wundermanthompson.com/service/technology)

[![Last Release Version](https://img.shields.io/github/v/release/wttech/aemc?color=lightblue&label=Last%20Release)](https://github.com/wttech/terraform-aem-provider/tags)
[![Ansible Galaxy](https://img.shields.io/ansible/collection/2218?label=Ansible%20Galaxy)](https://galaxy.ansible.com/wttech/aem)
[![Apache License, Version 2.0, January 2004](https://github.com/wttech/terraform-aem-provider/raw/main/images/apache-license-badge.svg)](http://www.apache.org/licenses/)

# AEM Compose - Terraform Provider

Allows to manage and provision Adobe Experience Manager (AEM) instances declaratively. 
Built on top of [AEM Compose](https://github.com/wttech/aemc).

## Example usage

```hcl
resource "aws_instance" "aem_author" {
  // ...
}

resource "aem_aws_instance" "aem_author" {
  aws {
    id  = aws_instance.aem.id
    ssm = true // prefer SSM over SSH when connecting to instance to provision it
  }

  config {
    instance_id = "local_author"
    file        = "aem.yml"  // or yml inline below

    inline = <<EOT
      instance:
        config:
          local_author:
            http_url: http://127.0.0.1:4502
            user: admin
            password: admin
            run_modes: [ int ]
            jvm_opts:
              - -server
              - -Djava.awt.headless=true
              - -Djava.io.tmpdir=[[canonicalPath .Path "aem/home/tmp"]]
              - -agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=0.0.0.0:14502
              - -Duser.language=en
              - -Duser.country=US
              - -Duser.timezone=UTC
            start_opts: []
            secret_vars:
              - ACME_SECRET=value
            env_vars:
              - ACME_VAR=value
            sling_props: []
      EOT
  }

  provision {
    commands = [
      // assumes usage of standard 'changed' field returned by AEMC
      ["pkg", "deploy", "--url", "http://github.com/../some-pkg.zip"],
      ["osgi", "config", "save", "--pid", "xxx", "props", "a: 'b'"]
    ]

    // nicely propagates 'changed' to TF (update in place), also automatically uploads packages to AEM
    packages = [
      "http://github.com/../some-pkg.zip",
      "packages/core-components.zip",
      "packages/content-large.zip" // use checksums to avoid re-uploading big packages
    ]

    // or as a last resort (without telling 'changed' to TF) 
    shell = <<EOT
        sh aemw pkg deploy --url "http://github.com/../some-pkg.zip"
        sh aemw [do ant
    EOT
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
