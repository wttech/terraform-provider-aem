This resource allows you to deploy an AEM package ZIP file on AEM instances when the CRX Package is not accessible from the public network (e.g., when accessed through a load balancer or application gateway).

The deployment process has been optimized for efficiency. If the same package is already deployed on the AEM instances, the provider will not re-upload the package file. This optimization significantly reduces network usage and streamlines the deployment process.

## Example usages

```hcl
resource "aem_package" "app" {
  file = "target/mysite.all-1.0.0-SNAPSHOT.zip"
  instance = aem_instance.aem_single
  // options = "-A"
}
```
