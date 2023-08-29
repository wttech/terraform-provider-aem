resource "aws_instance" "aem_single" {
  ami                         = "ami-043e06a423cbdca17" // RHEL 8
  instance_type               = "m5.xlarge"
  associate_public_ip_address = true
  tags                        = local.tags
  key_name                    = aws_key_pair.main.key_name
}

data "tls_public_key" "main" {
  private_key_pem = file("ec2-key.cer")
}

resource "aws_key_pair" "main" {
  key_name   = "${local.workspace}-example-tf"
  public_key = data.tls_public_key.main.public_key_openssh
  tags       = local.tags
}

output "instance_ip" {
  value = aws_instance.aem_single.public_ip
}
