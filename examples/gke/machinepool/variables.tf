variable "pool_name" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "replicas" {
  type = number
}

variable "machine_type" {
  type    = string
  default = "e2-standard-2"
}
