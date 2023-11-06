resource "aws_instance" "aem_single" {
  ami                  = "ami-043e06a423cbdca17" // RHEL 8
  instance_type        = "m5.xlarge"
  iam_instance_profile = aws_iam_instance_profile.aem_ec2.name
  tags                 = local.tags
}

resource "aws_iam_instance_profile" "aem_ec2" {
  name = "${local.workspace}_aem_ec2"
  role = aws_iam_role.aem_ec2.name
  tags = local.tags
}

resource "aws_iam_role" "aem_ec2" {
  name               = "${local.workspace}_aem_ec2"
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
  tags = local.tags
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.aem_ec2.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy_attachment" "s3" {
  role       = aws_iam_role.aem_ec2.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
}

output "instance_ip" {
  value = aws_instance.aem_single.public_ip
}
