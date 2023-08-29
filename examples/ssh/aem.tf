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
    data_dir    = "/data/aemc"
    version     = "1.4.1"
    lib_dir     = "lib"
    config_file = "aem.yml"
  }
}
