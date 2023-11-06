resource "aem_instance" "single" {
  depends_on = [aws_instance.aem_single]

  client {
    type = "ssh"
    settings = {
      host        = aws_instance.aem_single.public_ip
      port        = 22
      user        = "ec2-user"
      private_key_file = local.ssh_private_key # cannot be put into state as this is OS-dependent
    }
  }
  compose {
    version     = "1.5.8"
    data_dir    = "/home/ec2-user/aemc"
    lib_dir     = "aem/home/lib"
    config_file = "aem/default/etc/aem.yml"
  }
  hook {
    bootstrap = <<EOF
      #!/bin/sh
      sudo yum install -y unzip && \
      curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
      unzip awscliv2.zip && \
      sudo ./aws/install && \
      mkdir -p "/home/ec2-user/aemc/aem/home/lib" && \
      aws s3 cp --recursive --no-progress "s3://aemc/instance/classic/" "/home/ec2-user/aemc/aem/home/lib"
    EOF
    initialize = <<EOF
      #!/bin/sh
      # sh aemw instance backup restore
    EOF
    provision = <<EOF
      #!/bin/sh
      sh aemw osgi bundle install --url "https://github.com/neva-dev/felix-search-webconsole-plugin/releases/download/2.0.0/search-webconsole-plugin-2.0.0.jar" && \
      sh aemw osgi config save --pid "org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet" --input-string "alias: /crx/server" && \
      sh aemw package deploy --file "aem/home/lib/aem-service-pkg-6.5.*.0.zip"
    EOF
  }
}

output "aem_instances" {
  value = aem_instance.single.instances
}
