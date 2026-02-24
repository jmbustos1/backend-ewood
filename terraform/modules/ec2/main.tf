resource "aws_instance" "tcp_server" {
  ami                    = var.ami_id
  instance_type          = var.instance_type
  iam_instance_profile   = var.iam_instance_profile
  vpc_security_group_ids = [var.security_group]
  key_name               = var.key_name
  user_data              = var.user_data

  tags = {
    Name = "ec2-ewood"
  }
}
