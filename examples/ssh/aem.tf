resource "aem_instance" "author" {
  depends_on = [aws_instance.aem_author]

  client {
    type = "ssh"
    settings = {
      host        = aws_instance.aem_author.public_ip
      port        = 22
      user        = "ec2-user"
      private_key_file = local.ssh_private_key
    }
  }
  compose {
    data_dir    = "/data/aemc"
    version     = "1.4.1"
    lib_dir     = "lib"
    config_file = "aem.yml"
    instance_id = "local_author"
  }
}
