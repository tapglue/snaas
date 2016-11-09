resource "aws_iam_role" "ecs-agent" {
  name                = "ecs-agent"
  assume_role_policy  = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "ecs-agent" {
  name    = "ecs-agent"
  role    = "${aws_iam_role.ecs-agent.id}"
  policy  = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:CreateCluster",
        "ecs:DeregisterContainerInstance",
        "ecs:DiscoverPollEndpoint",
        "ecs:Poll",
        "ecs:RegisterContainerInstance",
        "ecs:StartTelemetrySession",
        "ecs:Submit*",
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_iam_role" "ecs-scheduler" {
  name                = "ecs-scheduler"
  assume_role_policy  = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ecs.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "ecs-scheduler" {
  name    = "ecs-scheduler"
  role    = "${aws_iam_role.ecs-scheduler.id}"
  policy  = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "elasticloadbalancing:Describe*",
        "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
        "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
        "ec2:Describe*",
        "ec2:AuthorizeSecurityGroupIngress"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_iam_role" "rds-monitoring" {
  name                = "rds-monitoring"
  assume_role_policy  = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ecs.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "rds-monitoring" {
  name    = "rds-monitoring"
  role    = "${aws_iam_role.rds-monitoring.id}"
  policy  = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
      {
          "Sid": "EnableCreationAndManagementOfRDSCloudwatchLogGroups",
          "Effect": "Allow",
          "Action": [
              "logs:CreateLogGroup",
              "logs:PutRetentionPolicy"
          ],
          "Resource": [
              "arn:aws:logs:*:*:log-group:RDS*"
          ]
      },
      {
          "Sid": "EnableCreationAndManagementOfRDSCloudwatchLogStreams",
          "Effect": "Allow",
          "Action": [
              "logs:CreateLogStream",
              "logs:PutLogEvents",
              "logs:DescribeLogStreams",
              "logs:GetLogEvents"
          ],
          "Resource": [
              "arn:aws:logs:*:*:log-group:RDS*:log-stream:*"
          ]
      }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "ecs-agent-profile" {
  name  = "ecs-agent-profile"
  roles = [
    "${aws_iam_role.ecs-agent.name}",
  ]
}

resource "aws_security_group" "perimeter" {
  description = "perimeter firewall rules"
  name        = "perimeter"
  vpc_id      = "${aws_vpc.env.id}"

  tags {
    Name = "perimeter"
  }
}

resource "aws_key_pair" "debug" {
  key_name    = "debug"
  public_key  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCuFsJxH52k7iI4mseWljlbQhwIfbpVPuDCTOBo6YtI7xL3f3jfme4fqziwt+iqavRW2MgGsgoYGITNYstZa5zzT4Zo6CTZ0XpeLYZrrXQOxXrXjesRA478bCsU4gpCrPiy5Uzw3e2d1HLF/deLjnmREshzqaEQKoL8tzG51esBTIna+M5aWD0AGPFotO3J2sFTRnbAIxeVj4bKWAfaE2+WG1MX1VemDGeGrHmW6UbPoymHOD7Y5c/F00Bv+Pgk5LwCyRCvEzMLbl2GHpEJd3vcouwEToyADlN1rXc+85SfVtlwS8F3fX6vqjQ/2fMzG4syaDEeUJLsBcE2glNIwDH/ debug"
}

resource "aws_iam_user" "state-change-sr" {
  name  = "state-change-sr"
  path  = "/"
}

resource "aws_iam_user_policy" "state-change-sr" {
  name  = "state-change-sr"
  user  = "${aws_iam_user.state-change-sr.name}"
  policy  = <<EOF
{
   "Version": "2012-10-17",
   "Statement":[{
      "Effect":"Allow",
      "Action": [
        "sqs:SendMessage",
        "sqs:ReceiveMessage",
        "sqs:GetQueueUrl"
      ],
      "Resource":"arn:aws:sqs:*:${var.account}:*-state-change"
      }
   ]
}
EOF
}

resource "aws_iam_access_key" "state-change-sr" {
  user  = "${aws_iam_user.state-change-sr.name}"
}

resource "aws_security_group_rule" "perimeter_grafana_out" {
  from_port                 = 3000
  to_port                   = 3000
  type                      = "egress"
  protocol                  = "tcp"
  security_group_id         = "${aws_security_group.perimeter.id}"
  source_security_group_id  = "${aws_security_group.platform.id}"
}

resource "aws_security_group_rule" "perimeter_https_in" {
  cidr_blocks = [
    "0.0.0.0/0"
  ]
  from_port         = 443
  to_port           = 443
  type              = "ingress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.perimeter.id}"
}

resource "aws_security_group_rule" "perimeter_http_out" {
  cidr_blocks       = [
    "0.0.0.0/0"
  ]
  from_port         = 80
  to_port           = 80
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.perimeter.id}"
}

resource "aws_security_group_rule" "perimeter_https_out" {
  cidr_blocks       = [
    "0.0.0.0/0"
  ]
  from_port         = 443
  to_port           = 443
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.perimeter.id}"
}

resource "aws_security_group_rule" "perimeter_service_out" {
  from_port                 = 8080
  to_port                   = 8085
  type                      = "egress"
  protocol                  = "tcp"
  security_group_id         = "${aws_security_group.perimeter.id}"
  source_security_group_id  = "${aws_security_group.platform.id}"
}

resource "aws_security_group_rule" "perimeter_ssh_in" {
  cidr_blocks       = [
    "0.0.0.0/0",
  ]
  from_port         = 22
  to_port           = 22
  type              = "ingress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.perimeter.id}"
}

resource "aws_security_group_rule" "perimeter_ssh_out" {
  cidr_blocks       = [
    "10.0.0.0/16",
  ]
  from_port         = 22
  to_port           = 22
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.perimeter.id}"
}

resource "aws_security_group" "platform" {
  description = "platform firewall rules"
  name        = "platform"
  vpc_id      = "${aws_vpc.env.id}"

  tags {
    Name = "platform"
  }
}

resource "aws_security_group_rule" "platform_grafana_in" {
  from_port                 = 3000
  to_port                   = 3000
  type                      = "ingress"
  protocol                  = "tcp"
  security_group_id         = "${aws_security_group.platform.id}"
  source_security_group_id  = "${aws_security_group.perimeter.id}"
}

resource "aws_security_group_rule" "platform_http_out" {
  cidr_blocks       = [
    "0.0.0.0/0"
  ]
  from_port         = 80
  to_port           = 80
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
}

resource "aws_security_group_rule" "platform_https_out" {
  cidr_blocks       = [
    "0.0.0.0/0"
  ]
  from_port         = 443
  to_port           = 443
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
}

resource "aws_security_group_rule" "platform_ntp_out" {
  cidr_blocks       = [
    "0.0.0.0/0"
  ]
  from_port                 = 123
  to_port                   = 123
  type                      = "egress"
  protocol                  = "udp"
  security_group_id         = "${aws_security_group.platform.id}"
}

resource "aws_security_group_rule" "platform_postgres_in" {
  from_port         = 5432
  to_port           = 5432
  type              = "ingress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
  self              = true
}

resource "aws_security_group_rule" "platform_postgres_out" {
  from_port         = 5432
  to_port           = 5432
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
  self              = true
}

resource "aws_security_group_rule" "platform_prometheus_in" {
  from_port         = 9000
  to_port           = 9100
  type              = "ingress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
  self              = true
}

resource "aws_security_group_rule" "platform_prometheus_out" {
  from_port         = 9000
  to_port           = 9100
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
  self              = true
}

resource "aws_security_group_rule" "platform_redis_in" {
  from_port         = 6379
  to_port           = 6379
  type              = "ingress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
  self              = true
}

resource "aws_security_group_rule" "platform_redis_out" {
  from_port         = 6379
  to_port           = 6379
  type              = "egress"
  protocol          = "tcp"
  security_group_id = "${aws_security_group.platform.id}"
  self              = true
}

resource "aws_security_group_rule" "platform_service_in" {
  from_port                 = 8080
  to_port                   = 8085
  type                      = "ingress"
  protocol                  = "tcp"
  security_group_id         = "${aws_security_group.platform.id}"
  source_security_group_id  = "${aws_security_group.perimeter.id}"
}

resource "aws_security_group_rule" "platform_ssh_in" {
  from_port                 = 22
  to_port                   = 22
  type                      = "ingress"
  protocol                  = "tcp"
  security_group_id         = "${aws_security_group.platform.id}"
  source_security_group_id  = "${aws_security_group.perimeter.id}"
}
