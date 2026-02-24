resource "aws_ssm_association" "ssm_agent" {
  name = "AWS-UpdateSSMAgent"

  targets {
    key    = "InstanceIds"
    values = [var.instance_id]
  }
}
