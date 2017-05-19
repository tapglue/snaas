provider "aws" {
  region = "${var.region}"
}

resource "aws_vpc" "env" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags {
    Name = "${var.env}"
  }
}

resource "aws_internet_gateway" "env" {
  vpc_id = "${aws_vpc.env.id}"

  tags {
    Name = "${var.env}"
  }
}

resource "aws_nat_gateway" "env" {
  allocation_id = "${aws_eip.nat.id}"

  depends_on = [
    "aws_internet_gateway.env",
  ]

  subnet_id = "${aws_subnet.perimeter-a.id}"
}

resource "aws_eip" "nat" {
  vpc = true
}

resource "aws_subnet" "perimeter-a" {
  availability_zone       = "${var.region}a"
  cidr_block              = "10.0.0.0/23"
  map_public_ip_on_launch = true
  vpc_id                  = "${aws_vpc.env.id}"

  tags {
    Name = "perimeter-a"
  }
}

resource "aws_subnet" "perimeter-b" {
  availability_zone       = "${var.region}b"
  cidr_block              = "10.0.8.0/23"
  map_public_ip_on_launch = true
  vpc_id                  = "${aws_vpc.env.id}"

  tags {
    Name = "perimeter-b"
  }
}

resource "aws_route_table" "perimeter" {
  vpc_id = "${aws_vpc.env.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.env.id}"
  }

  tags {
    Name = "perimeter"
  }
}

resource "aws_route_table_association" "perimeter-a" {
  route_table_id = "${aws_route_table.perimeter.id}"
  subnet_id      = "${aws_subnet.perimeter-a.id}"
}

resource "aws_route_table_association" "perimeter-b" {
  route_table_id = "${aws_route_table.perimeter.id}"
  subnet_id      = "${aws_subnet.perimeter-b.id}"
}

resource "aws_subnet" "platform-a" {
  availability_zone = "${var.region}a"
  cidr_block        = "10.0.2.0/23"
  vpc_id            = "${aws_vpc.env.id}"

  tags {
    Name = "platform-a"
  }
}

resource "aws_subnet" "platform-b" {
  availability_zone = "${var.region}b"
  cidr_block        = "10.0.10.0/23"
  vpc_id            = "${aws_vpc.env.id}"

  tags {
    Name = "platform-b"
  }
}

resource "aws_subnet" "platform-peering" {
  availability_zone = "${var.region}a"
  cidr_block        = "10.0.18.0/23"
  vpc_id            = "${aws_vpc.env.id}"

  tags {
    Name = "platform-peering"
  }
}

resource "aws_route_table" "platform" {
  vpc_id = "${aws_vpc.env.id}"

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = "${aws_nat_gateway.env.id}"
  }

  tags {
    Name = "platform"
  }
}

resource "aws_route_table_association" "platform-a" {
  route_table_id = "${aws_route_table.platform.id}"
  subnet_id      = "${aws_subnet.platform-a.id}"
}

resource "aws_route_table_association" "platform-b" {
  route_table_id = "${aws_route_table.platform.id}"
  subnet_id      = "${aws_subnet.platform-b.id}"
}

resource "aws_instance" "bastion" {
  ami           = "${var.ami_minimal["${var.region}"]}"
  instance_type = "t2.medium"
  key_name      = "${aws_key_pair.access.key_name}"

  vpc_security_group_ids = [
    "${aws_security_group.perimeter.id}",
  ]

  subnet_id = "${aws_subnet.perimeter-a.id}"

  tags {
    Name = "bastion"
  }
}

resource "aws_eip" "bastion" {
  instance = "${aws_instance.bastion.id}"
  vpc      = true
}
