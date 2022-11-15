data "google_service_account" "sa" {
  account_id = "tfc-k8s-cluster-${var.cluster_name}"
}

data "google_container_cluster" "cluster" {
  name = var.cluster_name
}

resource "google_container_node_pool" "node_pool" {
  provider = google-beta

  cluster = data.google_container_cluster.cluster.name

  name       = var.pool_name
  node_count = var.replicas

  node_config {
    preemptible  = true
    machine_type = var.machine_type


    service_account = data.google_service_account.sa.email
    oauth_scopes    = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}

# get the instances in the node pool
data "google_compute_instance_group" "ig" {
  self_link = google_container_node_pool.node_pool.instance_group_urls[0]
}