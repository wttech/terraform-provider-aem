resource "aem_instance" "author" {
  depends_on = [aws_instance.aem_author]

  client {
    type     = "aws_ssm"
    settings = {
      instance_id = aws_instance.aem_author.id
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
