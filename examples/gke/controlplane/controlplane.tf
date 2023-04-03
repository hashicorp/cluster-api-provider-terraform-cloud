# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "google_compute_zones" "available" {}

data "google_container_engine_versions" "supported" {
  location       = data.google_compute_zones.available.names[0]
  version_prefix = var.kubernetes_version
}

resource "google_service_account" "sa" {
  account_id   = "tfc-k8s-cluster-${var.cluster_name}"
  display_name = "Kubernetes provider SA"
}

resource "google_container_cluster" "cluster" {
  provider           = google-beta
  name               = var.cluster_name 
  location           = data.google_compute_zones.available.names[0]
  min_master_version = data.google_container_engine_versions.supported.latest_master_version

  # NOTE: We can't create a cluster with no node pool defined, but we want to use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
}

# Generate a kubeconfig file
locals {
  kubeconfig = {
    apiVersion = "v1"
    kind       = "Config"
    preferences = {
      colors = true
    }
    current-context = google_container_cluster.cluster.name
    contexts = [
      {
        name = google_container_cluster.cluster.name
        context = {
          cluster   = google_container_cluster.cluster.name
          user      = google_service_account.sa.email
          namespace = "default"
        }
      }
    ]
    clusters = [
      {
        name = google_container_cluster.cluster.name
        cluster = {
          server                     = "https://${google_container_cluster.cluster.endpoint}"
          certificate-authority-data = google_container_cluster.cluster.master_auth[0].cluster_ca_certificate
        }
      }
    ]
    users = [
      {
        name = google_service_account.sa.email
        user = {
          exec = {
            apiVersion         = "client.authentication.k8s.io/v1beta1"
            command            = "gke-gcloud-auth-plugin"
            interactiveMode    = "Never"
            provideClusterInfo = true
          }
        }
      }
    ]
  }
}