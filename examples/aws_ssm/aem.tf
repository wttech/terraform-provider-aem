resource "aem_instance" "single" {
  depends_on = [aws_instance.aem_single]

  client {
    type     = "aws_ssm"
    settings = {
      instance_id = aws_instance.aem_single.id
    }
  }
  compose {
    version     = "1.4.1"
    data_dir    = "/home/ec2-user/aemc"
    lib_dir     = "aem/home/lib"
    config_file = "aem/default/etc/aem.yml"
  }
}
