resource "aws_instance" "aem_author" {
  ami                  = "ami-043e06a423cbdca17" // RHEL 8
  instance_type        = "m5.xlarge"
  iam_instance_profile = aws_iam_instance_profile.ssm.name
  tags                 = local.tags
}

data "tls_public_key" "main" {
  private_key_pem = file("ec2-key.cer")
}

resource "aws_key_pair" "main" {
  key_name   = local.workspace
  public_key = data.tls_public_key.main.public_key_openssh
  tags       = local.tags
}


resource "aws_iam_instance_profile" "ssm" {
  name = "${local.workspace}_ssm_ec2"
  role = aws_iam_role.ssm.name
  tags = local.tags
}

resource "aws_iam_role" "ssm" {
  name               = "${local.workspace}_ssm"
  assume_role_policy = <<EOF
  {
    "Version": "2012-10-17",
    "Statement": {
      "Effect": "Allow",
      "Principal": {"Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  }
EOF
  tags               = local.tags
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.ssm.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

output "instance_ip" {
  value = aws_instance.aem_author.public_ip
}
