output "kubeconfig" {
  value = yamlencode(local.kubeconfig)
}

output "control_plane_endpoint_host" {
  value = google_container_cluster.cluster.endpoint
}

output "control_plane_endpoint_port" {
  value = 443
}