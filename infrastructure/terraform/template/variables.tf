variable "account" {
  default     = ""
  description = "AWS account identifier"
  type        = "string"
}

variable "ami_ecs_agent" {
  default = {
    "us-east-1"     = "ami-1924770e"
    "us-east-2"     = "ami-bd3e64d8"
    "us-west-1"     = "ami-7f004b1f"
    "us-west-2"     = "ami-56ed4936"
    "eu-west-1"     = "ami-c8337dbb"
    "eu-central-1"  = "ami-dd12ebb2"
  }
  description = "AMIs used for ecs agent"
  type        = "map"
}

variable "ami_minimal" {
  default = {
    "us-east-1"     = "ami-6d1c2007"
    "us-east-2"     = "ami-6a2d760f"
    "us-west-1"     = "ami-af4333cf"
    "us-west-2"     = "ami-d2c924b2"
    "eu-west-1"     = "ami-7abd0209"
    "eu-central-1"  = "ami-9bf712f4"
  }
  description = "AMIs used for auxiliary hosts"
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
  default     = {
    "gateway-http" = "34"
    "sims" = "34"
  }
  description = "Versions used for deployed services"
  type        = "map"
}
