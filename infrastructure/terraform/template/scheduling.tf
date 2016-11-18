resource "aws_ecs_service" "gateway-http" {
  cluster         = "${aws_ecs_cluster.service.id}"
  depends_on      = [
    "aws_iam_instance_profile.ecs-agent-profile",
    "aws_db_instance.service-master",
    "aws_elasticache_cluster.ratelimiter",
  ]
  deployment_maximum_percent          = 200
  deployment_minimum_healthy_percent  = 50
  desired_count   = 3
  iam_role        = "${aws_iam_role.ecs-scheduler.arn}"
  name            = "gateway-http"
  task_definition = "${aws_ecs_task_definition.gateway-http.arn}"

  load_balancer {
    container_name  = "gateway-http"
    container_port  = 8083
    elb_name        = "${aws_elb.gateway-http.id}"
  }
}

resource "aws_ecs_task_definition" "gateway-http" {
  family                = "gateway-http"
  container_definitions = <<EOF
[
  {
    "command": [
      "./gateway-http",
      "-aws.id", "${aws_iam_access_key.state-change-sr.id}",
      "-aws.secret", "${aws_iam_access_key.state-change-sr.secret}",
      "-aws.region", "${var.region}",
      "-postgres.url", "postgres://${var.pg_username}:${var.pg_password}@${aws_route53_record.service-master.fqdn}:5432/${var.pg_db_name}?connect_timeout=5&sslmode=require",
      "-redis.addr", "${aws_route53_record.ratelimiter-cache.fqdn}:6379",
      "-source", "sqs"
    ],
    "cpu": 1024,
    "dnsSearchDomains": [
      "${var.env}.${var.region}"
    ],
    "essential": true,
    "image": "tapglue/snaas:${var.version["gateway-http"]}",
    "logConfiguration": {
      "logDriver": "syslog"
    },
    "memory": 2048,
    "name": "gateway-http",
    "portMappings": [
      {
        "containerPort": 8083,
        "hostPort": 8083
      },
      {
        "containerPort": 9000,
        "hostPort": 9000
      }
    ],
    "readonlyRootFilesystem": true,
    "workingDirectory": "/tapglue/"
  }
]
EOF
}

resource "aws_ecs_service" "sims" {
  cluster         = "${aws_ecs_cluster.service.id}"
  depends_on      = [
    "aws_iam_instance_profile.ecs-agent-profile",
    "aws_db_instance.service-master",
  ]
  deployment_maximum_percent          = 200
  deployment_minimum_healthy_percent  = 50
  desired_count   = 2
  name            = "sims"
  task_definition = "${aws_ecs_task_definition.sims.arn}"
}

resource "aws_ecs_task_definition" "sims" {
  family                = "sims"
  container_definitions = <<EOF
[
  {
    "command": [
      "./sims",
      "-aws.id", "${aws_iam_access_key.state-change-sr.id}",
      "-aws.secret", "${aws_iam_access_key.state-change-sr.secret}",
      "-aws.region", "${var.region}",
      "-postgres.url", "postgres://${var.pg_username}:${var.pg_password}@${aws_route53_record.service-master.fqdn}:5432/${var.pg_db_name}?connect_timeout=5&sslmode=require"
    ],
    "cpu": 256,
    "dnsSearchDomains": [
      "${var.env}.${var.region}"
    ],
    "essential": true,
    "image": "tapglue/snaas:${var.version["sims"]}",
    "logConfiguration": {
      "logDriver": "syslog"
    },
    "memory": 512,
    "name": "gateway-http",
    "portMappings": [
      {
        "containerPort": 9001,
        "hostPort": 9001
      }
    ],
    "readonlyRootFilesystem": true,
    "workingDirectory": "/tapglue/"
  }
]
EOF
}
