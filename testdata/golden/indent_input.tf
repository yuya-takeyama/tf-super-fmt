resource "aws_instance" "web" {
  ami           = "abc123"
  instance_type = "t2.micro"

  network_interface {
    device_index         = 0
    network_interface_id = aws_network_interface.foo.id
  }
}
