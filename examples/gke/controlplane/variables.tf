# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "cluster_name" {
  type = string
} 

variable "kubernetes_version" {
  type = string
}

variable "cluster_network" {
  type = object({})
}