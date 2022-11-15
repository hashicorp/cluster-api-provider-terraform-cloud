variable "cluster_name" {
  type = string
} 

variable "kubernetes_version" {
  type = string
}

variable "cluster_network" {
  type = object({})
}