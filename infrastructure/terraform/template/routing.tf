resource "aws_route53_zone" "env" {
  comment = "zone to isolate DNS routes for ${var.env}.${var.region}"
  name    = "${var.env}.${var.region}"
  vpc_id  = "${aws_vpc.env.id}"
}

resource "aws_route53_record" "ratelimiter-cache" {
  name    = "cache.ratelimiter"
  ttl     = "5"
  type    = "CNAME"
  zone_id = "${aws_route53_zone.env.id}"

  records = [
    "${aws_elasticache_cluster.ratelimiter.cache_nodes.0.address}",
  ]
}

resource "aws_route53_record" "service-master" {
  name    = "db-master.service"
  ttl     = "5"
  type    = "CNAME"
  zone_id = "${aws_route53_zone.env.id}"

  records = [
    "${aws_db_instance.service-master.address}",
  ]
}

resource "aws_route53_zone" "perimeter" {
  comment = "Zone for public termination of the env."
  name    = "${replace(var.domain, "*.", "")}"
}

resource "aws_route53_record" "console" {
  name    = "console-${var.env}-${var.region}"
  ttl     = "60"
  type    = "CNAME"
  zone_id = "${aws_route53_zone.perimeter.id}"

  records = [
    "${aws_elb.console.dns_name}",
  ]
}

resource "aws_route53_record" "gateway-http" {
  name    = "${var.env}-${var.region}"
  ttl     = "60"
  type    = "CNAME"
  zone_id = "${aws_route53_zone.perimeter.id}"

  records = [
    "${aws_elb.gateway-http.dns_name}",
  ]
}

resource "aws_route53_record" "monitoring" {
  name    = "monitoring-${var.env}-${var.region}"
  ttl     = "60"
  type    = "CNAME"
  zone_id = "${aws_route53_zone.perimeter.id}"

  records = [
    "${aws_elb.monitoring.dns_name}",
  ]
}
