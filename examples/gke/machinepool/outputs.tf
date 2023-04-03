# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "provider_id_list" {
  value = data.google_compute_instance_group.ig.instances
}