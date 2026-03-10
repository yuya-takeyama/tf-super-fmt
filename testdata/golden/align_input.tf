resource "aws_instance" "web" {
  ami = "abc123"
  instance_type = "t2.micro"
  tags = local.common_tags
}
