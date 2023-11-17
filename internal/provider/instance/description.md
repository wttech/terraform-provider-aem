The instance resource allows you to create and manage AEM instances.

With this resource, you can set up one or many AEM instances on a single machine. 

If you need to set up multiple AEM instances on multiple machines, you can use this resource multiple times. However, remember to use different client settings and adapt the compose configuration accordingly. This is because the default configuration assumes that both AEM author and publish are set up on the same machine.

## Example usages

Consider reviewing the following examples to find the one that best suits your needs:

1. [AWS EC2 instance with public IP](https://github.com/wttech/terraform-provider-aem/tree/main/examples/aws_ssh)
2. [AWS EC2 instance with private IP](https://github.com/wttech/terraform-provider-aem/tree/main/examples/aws_ssm)
3. [Bare metal machine](https://github.com/wttech/terraform-provider-aem/tree/main/examples/bare_metal_ssh)
