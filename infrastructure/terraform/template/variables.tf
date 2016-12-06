variable "account" {
  default     = ""
  description = "AWS account identifier"
  type        = "string"
}

variable "ami_ecs_agent" {
  default = {
    "us-east-1"    = "ami-1924770e"
    "us-east-2"    = "ami-bd3e64d8"
    "us-west-1"    = "ami-7f004b1f"
    "us-west-2"    = "ami-56ed4936"
    "eu-west-1"    = "ami-c8337dbb"
    "eu-central-1" = "ami-dd12ebb2"
  }

  description = "AMIs used for ecs agent"
  type        = "map"
}

variable "ami_minimal" {
  default = {
    "us-east-1"    = "ami-1924770e"
    "us-east-2"    = "ami-bd3e64d8"
    "us-west-1"    = "ami-7f004b1f"
    "us-west-2"    = "ami-56ed4936"
    "eu-west-1"    = "ami-c8337dbb"
    "eu-central-1" = "ami-dd12ebb2"
  }

  description = "AMIs used for auxiliary hosts"
  type        = "map"
}

variable "domain" {
  default     = ""
  description = "Domain for public termination of the env."
  type        = "string"
}

variable "elb_id" {
  default = {
    "us-east-1"      = "127311923021"
    "us-east-2"      = "033677994240"
    "us-west-1"      = "027434742980"
    "us-west-2"      = "797873946194"
    "eu-west-1"      = "156460612806"
    "eu-central-1"   = "054676820928"
    "ap-northeast-1" = "582318560864"
    "ap-northeast-2" = "600734575887"
    "ap-southeast-1" = "114774131450"
    "ap-southeast-2" = "783225319266"
    "ap-south-1"     = "718504428378"
    "sa-east-1"      = "507241528517"
    "us-gov-west-1"  = "048591011584"
    "cn-north-1"     = "638102146993"
  }

  description = "Mapping of ELB account IDs needed to enable log access on S3 bucket"
  type        = "map"
}

variable "env" {
  default     = ""
  description = "environment name used for isolation"
  type        = "string"
}

variable "key" {
  default = {
    "access" = ""
  }

  description = "SSH public keys"
  type        = "map"
}

variable "region" {
  default     = ""
  description = "Region to deploy to"
  type        = "string"
}

variable "pg_db_name" {
  default     = "tapglue"
  description = "Postgres database name"
  type        = "string"
}

variable "pg_username" {
  default     = "tapglue"
  description = "Postgres database username"
  type        = "string"
}

variable "pg_password" {
  default     = ""
  description = "Postgres database password"
  type        = "string"
}

variable "version" {
  default = {
    "gateway-http" = "168"
    "sims"         = "168"
  }

  description = "Versions used for deployed services"
  type        = "map"
}
