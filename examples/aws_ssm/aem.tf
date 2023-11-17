resource "aem_instance" "single" {
  depends_on = [aws_instance.aem_single, aws_volume_attachment.aem_single_data]

  client {
    type = "aws_ssm"
    settings = {
      instance_id = aws_instance.aem_single.id
      region      = "eu-central-1" // TODO infer from AWS provider config
    }
  }

  system {
    data_dir = local.aem_single_compose_dir
    bootstrap = {
      inline = [
        // mounting AWS EBS volume into data directory
        "sudo mkfs -t ext4 ${local.aem_single_data_device}",
        "sudo mkdir -p ${local.aem_single_data_dir}",
        "sudo mount ${local.aem_single_data_device} ${local.aem_single_data_dir}",
        "sudo chown -R ${local.ssh_user} ${local.aem_single_data_dir}",
        "echo '${local.aem_single_data_device} ${local.aem_single_data_dir} ext4 defaults 0 0' | sudo tee -a /etc/fstab",
        // installing AWS CLI: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html
        "sudo yum install -y unzip",
        "curl 'https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip' -o 'awscliv2.zip'",
        "unzip -q awscliv2.zip",
        "sudo ./aws/install --update",
        // installing AWS SSM agent: https://docs.aws.amazon.com/systems-manager/latest/userguide/agent-install-rhel-8-9.html
        "sudo dnf install -y https://s3.amazonaws.com/ec2-downloads-windows/SSMAgent/latest/linux_amd64/amazon-ssm-agent.rpm",
      ]
    }
  }

  compose {
    config = file("aem.yml") // use templating here if needed: https://developer.hashicorp.com/terraform/language/functions/templatefile
    create = {
      inline = [
        "mkdir -p '${local.aem_single_compose_dir}/aem/home/lib'",
        "aws s3 cp --recursive --no-progress 's3://aemc/instance/classic/' '${local.aem_single_compose_dir}/aem/home/lib'",
        "sh aemw instance init",
        "sh aemw instance create",
      ]
    }
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
