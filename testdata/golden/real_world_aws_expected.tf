terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

// This module creates the VPC for the project
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.8.1"

  name = "${var.project}-vpc"
  cidr = var.vpc_cidr

  azs             = var.azs
  public_subnets  = var.public_subnets
  private_subnets = var.private_subnets

  enable_nat_gateway = true
  single_nat_gateway = true

  tags = merge(var.common_tags, {
    Name = "${var.project}-vpc"
  })
}

resource "aws_security_group" "web" {
  name        = "${var.project}-web-sg"
  description = "Allow HTTP/HTTPS/SSH"
  vpc_id      = module.vpc.vpc_id

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidrs
  }

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidrs
  }

  # SSH is temporarily allowed for operational reasons
  ingress {
    description = "SSH temporary"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.admin_cidrs
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.common_tags, { Name = "${var.project}-web-sg" })
}

resource "aws_instance" "web" {
  ami           = var.ami_id
  instance_type = var.instance_type
  key_name      = var.key_name

  # Use network_interface block explicitly
  network_interface {
    network_interface_id = aws_network_interface.web_eni.id
    device_index         = 0
  }

  user_data = <<-EOT
    #!/bin/bash
    set -eux
    echo "project=${var.project}" > /etc/project-info
    dnf -y install nginx
    systemctl enable nginx
    systemctl start nginx
    EOT

  lifecycle {
    ignore_changes = [
      # Ignore tag changes from SSM patch management
      tags["PatchWindow"],
      user_data,
    ]
    create_before_destroy = true
  }

  tags = merge(var.common_tags, {
    Name = "${var.project}-web-ec2"
    Role = "web"
  })
}
