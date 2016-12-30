data "aws_acm_certificate" "perimeter" {
  domain = "${var.domain}"

  statuses = [
    "ISSUED",
  ]
}

data "template_file" "domain_canonical" {
  template = "$${root}.$${tld}"

  vars {
    root = "${element(split(".", var.domain), length(split(".", var.domain)) - 2)}"
    tld  = "${element(split(".", var.domain), length(split(".", var.domain)) - 1)}"
  }
}

data "template_file" "monitoring-user_data" {
  template = "${file("${path.module}/scripts/setup_monitoring.sh")}"

  vars {
    aws_id               = "${aws_iam_access_key.monitoring.id}"
    aws_secret           = "${aws_iam_access_key.monitoring.secret}"
    dashboard            = "${file("${path.module}/files/dashboard-ops.json")}"
    domain               = "${replace(var.domain, "*.", "")}"
    domain_canonical     = "${data.template_file.domain_canonical.rendered}"
    google_client_id     = "${var.google_client_id}"
    google_client_secret = "${var.google_client_secret}"
    region               = "${var.region}"
    zone                 = "${var.env}-${var.region}"
  }
}

resource "aws_instance" "monitoring" {
  ami           = "${var.ami_minimal["${var.region}"]}"
  instance_type = "t2.medium"
  key_name      = "${aws_key_pair.access.key_name}"

  vpc_security_group_ids = [
    "${aws_security_group.platform.id}",
  ]

  subnet_id = "${aws_subnet.platform-a.id}"

  tags {
    Name = "monitoring"
  }

  provisioner "file" {
    connection {
      type                = "ssh"
      user                = "admin"
      private_key         = "${file("${path.cwd}/access.pem")}"
      bastion_user        = "admin"
      bastion_private_key = "${file("${path.cwd}/access.pem")}"
      bastion_host        = "${aws_eip.bastion.public_ip}"
    }

    destination = "/tmp/setup.sh"
    content     = "${data.template_file.monitoring-user_data.rendered}"
  }

  provisioner "remote-exec" {
    connection {
      type                = "ssh"
      user                = "admin"
      private_key         = "${file("${path.cwd}/access.pem")}"
      bastion_user        = "admin"
      bastion_private_key = "${file("${path.cwd}/access.pem")}"
      bastion_host        = "${aws_eip.bastion.public_ip}"
    }

    inline = [
      "chmod +x /tmp/setup.sh",
      "sudo /tmp/setup.sh",
    ]
  }
}

resource "aws_elb" "monitoring" {
  connection_draining         = true
  connection_draining_timeout = 10
  cross_zone_load_balancing   = true
  idle_timeout                = 30
  name                        = "monitoring"

  security_groups = [
    "${aws_security_group.perimeter.id}",
  ]

  subnets = [
    "${aws_subnet.perimeter-a.id}",
    "${aws_subnet.perimeter-b.id}",
  ]

  access_logs = {
    bucket        = "${aws_s3_bucket.logs-elb.id}"
    bucket_prefix = "monitoring"
    interval      = 5
  }

  instances = [
    "${aws_instance.monitoring.id}",
  ]

  listener {
    instance_port      = 3000
    instance_protocol  = "http"
    lb_port            = 443
    lb_protocol        = "https"
    ssl_certificate_id = "${data.aws_acm_certificate.perimeter.arn}"
  }

  tags {
    Name = "monitoring"
  }
}

resource "aws_autoscaling_group" "service" {
  desired_capacity          = 3
  health_check_grace_period = 60
  health_check_type         = "EC2"
  launch_configuration      = "${aws_launch_configuration.service.name}"

  load_balancers = [
    "${aws_elb.gateway-http.name}",
  ]

  max_size = 30
  min_size = 1
  name     = "service"

  termination_policies = [
    "OldestInstance",
    "OldestLaunchConfiguration",
    "ClosestToNextInstanceHour",
  ]

  vpc_zone_identifier = [
    "${aws_subnet.platform-a.id}",
    "${aws_subnet.platform-b.id}",
  ]

  tag {
    key                 = "Name"
    value               = "service"
    propagate_at_launch = true
  }
}

resource "aws_launch_configuration" "service" {
  associate_public_ip_address = false
  ebs_optimized               = false
  enable_monitoring           = true
  key_name                    = "${aws_key_pair.access.key_name}"
  iam_instance_profile        = "${aws_iam_instance_profile.ecs-agent-profile.name}"
  image_id                    = "${var.ami_ecs_agent["${var.region}"]}"
  instance_type               = "m4.large"
  name_prefix                 = "ecs-service-"

  security_groups = [
    "${aws_security_group.platform.id}",
  ]

  lifecycle {
    create_before_destroy = true
  }

  user_data = <<EOF
#!/bin/bash
echo ECS_CLUSTER=service >> /etc/ecs/ecs.config

# Rsyslog tooling
sudo yum install -y rsyslog-gnutls
sudo mkdir -p /var/spool/rsyslog

sudo service rsyslog restart

# Rotate logs frequentely.
echo '#!/bin/sh

/usr/sbin/logrotate /etc/logrotate.hourly.conf >/dev/null 2>&1
EXITVALUE=$?
if [ $EXITVALUE != 0 ]; then
    /usr/bin/logger -t logrotate "ALERT exited abnormally with [$EXITVALUE]"
fi
exit 0
' | sudo tee /etc/cron.hourly/logrotate > /dev/null

sudo chmod +x /etc/cron.hourly/logrotate

echo '/var/log/messages {
    compress
    create
    daily
    rotate 5
    size 100M
    postrotate
  /bin/kill -HUP `cat /var/run/syslogd.pid 2> /dev/null` 2> /dev/null || true
    endscript
}' | sudo tee /etc/logrotate.hourly.conf > /dev/null

EOF
}

resource "aws_elb" "gateway-http" {
  connection_draining         = true
  connection_draining_timeout = 10
  cross_zone_load_balancing   = true
  idle_timeout                = 30
  name                        = "gateway-http"

  security_groups = [
    "${aws_security_group.perimeter.id}",
  ]

  subnets = [
    "${aws_subnet.perimeter-a.id}",
    "${aws_subnet.perimeter-b.id}",
  ]

  access_logs = {
    bucket        = "${aws_s3_bucket.logs-elb.id}"
    bucket_prefix = "gateway-http"
    interval      = 5
  }

  health_check {
    healthy_threshold   = 2
    interval            = 5
    target              = "HTTP:8083/health-45016490610398192"
    timeout             = 2
    unhealthy_threshold = 2
  }

  listener {
    instance_port      = 8083
    instance_protocol  = "http"
    lb_port            = 443
    lb_protocol        = "https"
    ssl_certificate_id = "${data.aws_acm_certificate.perimeter.arn}"
  }

  tags {
    Name = "gateway-http"
  }
}

resource "aws_ecs_cluster" "service" {
  name = "service"
}
