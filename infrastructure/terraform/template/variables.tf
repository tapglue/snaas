variable "account" {
  default     = ""
  description = "AWS account identifier"
  type        = "string"
}

variable "ami_ecs_agent" {
  default = {
    "ap-northeast-1" = "ami-08f7956f"
    "ap-southeast-1" = "ami-f4832f97"
    "ap-southeast-2" = "ami-774b7314"
    "ca-central-1"   = "ami-be45f7da"
    "eu-central-1"   = "ami-dd12ebb2"
    "eu-west-1"      = "ami-c8337dbb"
    "us-east-1"      = "ami-1924770e"
    "us-east-2"      = "ami-bd3e64d8"
    "us-west-1"      = "ami-7f004b1f"
    "us-west-2"      = "ami-56ed4936"
  }

  description = "AMIs used for ecs agent"
  type        = "map"
}

variable "ami_minimal" {
  default = {
    "ap-northeast-1" = "ami-50ed4631"
    "ap-northeast-2" = "ami-8e6abee0"
    "ap-south-1"     = "ami-c5e490aa"
    "ap-southeast-1" = "ami-0e6dce6d"
    "ap-southeast-2" = "ami-9cc6f9ff"
    "eu-central-1"   = "ami-cc8441a3"
    "eu-west-1"      = "ami-7d45150e"
    "sa-east-1"      = "ami-3b41de57"
    "us-east-1"      = "ami-49e5cb5e"
    "us-east-2"      = "ami-0e79236b"
    "us-west-1"      = "ami-db6c39bb"
    "us-west-2"      = "ami-8f7bd9ef"
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
    "ap-northeast-1" = "582318560864"
    "ap-northeast-2" = "600734575887"
    "ap-south-1"     = "718504428378"
    "ap-southeast-1" = "114774131450"
    "ap-southeast-2" = "783225319266"
    "cn-north-1"     = "638102146993"
    "eu-central-1"   = "054676820928"
    "eu-west-1"      = "156460612806"
    "sa-east-1"      = "507241528517"
    "us-east-1"      = "127311923021"
    "us-east-2"      = "033677994240"
    "us-gov-west-1"  = "048591011584"
    "us-west-1"      = "027434742980"
    "us-west-2"      = "797873946194"
  }

  description = "Mapping of ELB account IDs needed to enable log access on S3 bucket"
  type        = "map"
}

variable "env" {
  default     = ""
  description = "environment name used for isolation"
  type        = "string"
}

variable "google_client_id" {
  default     = ""
  description = "Client id for google OAuth"
  type        = "string"
}

variable "google_client_secret" {
  default     = ""
  description = "Client secret for google OAuth"
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
    "console"      = "351"
    "gateway-http" = "351"
    "sims"         = "351"
  }

  description = "Versions used for deployed services"
  type        = "map"
}
