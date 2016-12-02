resource "aws_db_subnet_group" "service" {
  description = "Service Postgres"
  name        = "service-${var.env}-${var.region}"

  subnet_ids = [
    "${aws_subnet.platform-a.id}",
    "${aws_subnet.platform-b.id}",
  ]
}

resource "aws_db_parameter_group" "service-master" {
  description = "Service Postgres master"
  family      = "postgres9.5"
  name        = "service-master"

  parameter {
    apply_method = "pending-reboot"
    name         = "log_statement"
    value        = "all"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "log_min_duration_statement"
    value        = "20"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "log_duration"
    value        = "1"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "shared_preload_libraries"
    value        = "pg_stat_statements"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "track_activity_query_size"
    value        = "2048"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "pg_stat_statements.track"
    value        = "ALL"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "autovacuum"
    value        = "1"
  }

  parameter {
    apply_method = "immediate"
    name         = "autovacuum_naptime"
    value        = "300"
  }

  parameter {
    apply_method = "immediate"
    name         = "autovacuum_vacuum_scale_factor"
    value        = "0.2"
  }

  parameter {
    apply_method = "immediate"
    name         = "autovacuum_vacuum_threshold"
    value        = "5000"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "log_autovacuum_min_duration"
    value        = "1"
  }

  parameter {
    apply_method = "immediate"
    name         = "maintenance_work_mem"
    value        = "768000"
  }

  parameter {
    apply_method = "pending-reboot"
    name         = "max_connections"
    value        = "256"
  }

  parameter {
    apply_method = "immediate"
    name         = "standard_conforming_strings"
    value        = "1"
  }

  parameter {
    apply_method = "immediate"
    name         = "work_mem"
    value        = "64000"
  }
}

resource "aws_db_instance" "service-master" {
  allocated_storage         = "300"
  apply_immediately         = true
  backup_retention_period   = 30
  backup_window             = "04:00-04:30"
  db_subnet_group_name      = "${aws_db_subnet_group.service.id}"
  final_snapshot_identifier = "service-master-${var.env}-${var.region}-final"
  identifier                = "service-master"
  iops                      = 3000
  storage_type              = "io1"
  engine                    = "postgres"
  engine_version            = "9.5.4"
  instance_class            = "db.r3.xlarge"
  maintenance_window        = "sat:05:00-sat:06:30"

  monitoring_interval = 1
  monitoring_role_arn = "${aws_iam_role.rds-monitoring.arn}"
  multi_az            = true

  parameter_group_name = "${aws_db_parameter_group.service-master.id}"
  publicly_accessible  = false
  skip_final_snapshot  = false
  storage_encrypted    = true

  vpc_security_group_ids = [
    "${aws_security_group.platform.id}",
  ]

  name     = "${var.pg_db_name}"
  username = "${var.pg_username}"
  password = "${var.pg_password}"
}

resource "aws_elasticache_subnet_group" "ratelimiter" {
  description = "ratelimiter cache"
  name        = "ratelimiter"

  subnet_ids = [
    "${aws_subnet.platform-a.id}",
    "${aws_subnet.platform-b.id}",
  ]
}

resource "aws_elasticache_cluster" "ratelimiter" {
  cluster_id           = "ratelimiter"
  engine               = "redis"
  engine_version       = "2.8.21"
  maintenance_window   = "sun:05:00-sun:06:00"
  node_type            = "cache.t2.micro"
  num_cache_nodes      = 1
  parameter_group_name = "default.redis2.8"
  port                 = 6379

  security_group_ids = [
    "${aws_security_group.platform.id}",
  ]

  subnet_group_name = "${aws_elasticache_subnet_group.ratelimiter.name}"
}

resource "aws_s3_bucket" "logs-elb" {
  bucket        = "${var.account}-snaas-${var.region}-${var.env}-logs-elb"
  force_destroy = true

  policy = <<EOF
{
	"Version": "2012-10-17",
	"Id": "Policy1458936351610",
	"Statement": [
		{
			"Sid": "Stmt1458936348932",
			"Effect": "Allow",
			"Principal": {
				"AWS": "arn:aws:iam::${var.elb_id["${var.region}"]}:root"
			},
			"Action": "s3:PutObject",
			"Resource": "arn:aws:s3:::${var.account}-snaas-${var.region}-${var.env}-logs-elb/*"
		}
	]
}
EOF
}

resource "aws_sqs_queue" "connection-state-change-dlq" {
  delay_seconds              = 0
  max_message_size           = 262144
  message_retention_seconds  = 1209600
  name                       = "connection-state-change-dlq"
  receive_wait_time_seconds  = 1
  visibility_timeout_seconds = 300
}

resource "aws_sqs_queue" "connection-state-change" {
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 1209600
  name                      = "connection-state-change"
  receive_wait_time_seconds = 1

  redrive_policy = <<EOF
{
    "deadLetterTargetArn": "${aws_sqs_queue.connection-state-change-dlq.arn}",
    "maxReceiveCount": 10
}
EOF

  visibility_timeout_seconds = 60
}

resource "aws_sqs_queue" "event-state-change-dlq" {
  delay_seconds              = 0
  max_message_size           = 262144
  message_retention_seconds  = 1209600
  name                       = "event-state-change-dlq"
  receive_wait_time_seconds  = 1
  visibility_timeout_seconds = 300
}

resource "aws_sqs_queue" "event-state-change" {
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 1209600
  name                      = "event-state-change"
  receive_wait_time_seconds = 1

  redrive_policy = <<EOF
{
    "deadLetterTargetArn": "${aws_sqs_queue.event-state-change-dlq.arn}",
    "maxReceiveCount": 10
}
EOF

  visibility_timeout_seconds = 60
}

resource "aws_sqs_queue" "object-state-change-dlq" {
  delay_seconds              = 0
  max_message_size           = 262144
  message_retention_seconds  = 1209600
  name                       = "object-state-change-dlq"
  receive_wait_time_seconds  = 1
  visibility_timeout_seconds = 300
}

resource "aws_sqs_queue" "object-state-change" {
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 1209600
  name                      = "object-state-change"
  receive_wait_time_seconds = 1

  redrive_policy = <<EOF
{
    "deadLetterTargetArn": "${aws_sqs_queue.object-state-change-dlq.arn}",
    "maxReceiveCount": 10
}
EOF

  visibility_timeout_seconds = 60
}

# Device update queues, topics and subscriptions.
resource "aws_sqs_queue" "endpoint-state-change-dlq" {
  delay_seconds              = 0
  max_message_size           = 262144
  message_retention_seconds  = 1209600
  name                       = "endpoint-state-change-dlq"
  receive_wait_time_seconds  = 1
  visibility_timeout_seconds = 300
}

resource "aws_sqs_queue" "endpoint-state-change" {
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 1209600
  name                      = "endpoint-state-change"
  receive_wait_time_seconds = 1

  redrive_policy = <<EOF
{
    "deadLetterTargetArn": "${aws_sqs_queue.endpoint-state-change-dlq.arn}",
    "maxReceiveCount": 10
}
EOF

  visibility_timeout_seconds = 60
}

resource "aws_sns_topic" "endpoint-state-change" {
  name = "endpoint-state-change"
}

resource "aws_sns_topic_subscription" "endpoint-state-change" {
  endpoint  = "${aws_sqs_queue.endpoint-state-change.arn}"
  protocol  = "sqs"
  topic_arn = "${aws_sns_topic.endpoint-state-change.arn}"
}
