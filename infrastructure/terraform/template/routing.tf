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
