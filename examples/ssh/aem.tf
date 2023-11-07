resource "aem_instance" "single" {
  depends_on = [aws_instance.aem_single, aws_volume_attachment.aem_single_data]

  client {
    type     = "ssh"
    settings = {
      host             = aws_instance.aem_single.public_ip
      port             = 22
      user             = local.ssh_user
      private_key_file = local.ssh_private_key # cannot be put into state as this is OS-dependent
    }
  }
  /* TODO support this as well
  files = {
    "lib" = "${local.aem_single_compose_dir}/aem/home/lib"
  }
  */
  system {
    data_dir  = local.aem_single_compose_dir
    bootstrap = <<EOF
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
    EOF
  }
  compose {
    version    = "1.5.8"
    launch = <<EOF
      #!/bin/sh
      sh aemw osgi bundle install --url "https://github.com/neva-dev/felix-search-webconsole-plugin/releases/download/2.0.0/search-webconsole-plugin-2.0.0.jar" && \
      sh aemw osgi config save --pid "org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet" --input-string "alias: /crx/server" && \
      echo "
      enabled: true
      transportUri: http://localhost:4503/bin/receive?sling:authRequestLogin=1
      transportUser: admin
      transportPassword: admin
      userId: admin
      " | sh aemw repl agent setup -A --location "author" --name "publish" && \
      sh aemw package deploy --file "aem/home/lib/aem-service-pkg-6.5.*.0.zip"
    EOF
  }
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
