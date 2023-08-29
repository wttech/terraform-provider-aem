resource "aws_instance" "aem_single" {
  ami                         = "ami-043e06a423cbdca17" // RHEL 8
  instance_type               = "m5.xlarge"
  associate_public_ip_address = true
  tags                        = local.tags
}

output "instance_ip" {
  value = aws_instance.aem_single.public_ip
}
