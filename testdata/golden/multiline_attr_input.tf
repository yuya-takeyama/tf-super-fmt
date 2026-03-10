resource "aws_instance" "web" {
  tags = merge(var.common_tags, {
    Name = "web"
    Role = "frontend"
  })
}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

resource "foo" "bar" {
  lifecycle {
    ignore_changes = [
      tags,
      user_data,
    ]
  }
}
