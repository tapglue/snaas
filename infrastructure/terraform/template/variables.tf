variable "account" {
  default     = ""
  description = "AWS account identifier"
  type        = "string"
}

variable "ami" {
  default     = {}
  description = "AMIs used for components"
  type        = "map"
}

variable "env" {
  default     = ""
  description = "environment name used for isolation"
  type        = "string"
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
  default     = {}
  description = "Versions used for deployed services"
  type        = "map"
}
