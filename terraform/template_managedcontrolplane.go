package terraform

const ManagedClusterConfigurationTemplate = `
{{- if .Object.Spec.Variables }}
  {{ range $v := .Object.Spec.Variables }}
variable "{{ $v.Name }}" {}
  {{- end}}
{{- end }}

module "cluster" {
  source = "{{ .Object.Spec.Module.Source }}"
{{- if .Object.Spec.Module.Version }}
  version = "{{ .Object.Spec.Module.Version }}"
{{- end }}
{{- if .Object.Spec.Variables }}
  {{ range $v := .Object.Spec.Variables }}
    {{ $v.Name }} = var.{{ $v.Name }}
  {{- end }}
{{- end }}

  cluster_name       = "{{ .Owner.ObjectMeta.Name }}"
  kubernetes_version = "{{ .Object.Spec.Version }}"

{{- if .Owner.Spec.ClusterNetwork }}
  cluster_network = {
  {{ if .Owner.Spec.ClusterNetwork.APIServerPort }}
    api_server_port = {{ .Owner.Spec.ClusterNetwork.APIServerPort }}
  {{- end }}

  {{ if .Owner.Spec.ClusterNetwork.ServiceDomain }}
    service_domain = {{ .Owner.Spec.ClusterNetwork.ServiceDomain }}
  {{- end }}

  {{ if .Owner.Spec.ClusterNetwork.Pods }}
    cidr_blocks = [
	  {{ range $v := .Owner.Spec.ClusterNetwork.Pods.CIDRBlocks }}
      "{{ $v }}",
	  {{- end}}
	]
  {{- end }}

  {{ if .Owner.Spec.ClusterNetwork.Services }}
    cidr_blocks = [
	  {{ range $v := .Owner.Spec.ClusterNetwork.Services.CIDRBlocks }}
      "{{ $v }}",
	  {{- end}}
	]
  {{- end }}
  }
{{- end }}
}

output "control_plane_endpoint_host" {
  value = module.cluster.control_plane_endpoint_host
}

output "control_plane_endpoint_port" {
  value = module.cluster.control_plane_endpoint_port
}

output "kubeconfig" {
	value = module.cluster.kubeconfig
  }
`
