package terraform

const ManagedMachinePoolConfigurationTemplate = `
{{- if .Object.Spec.Variables }}
  {{ range $v := .Object.Spec.Variables }}
variable "{{ $v.Name }}" {}
  {{- end}}
{{- end }}

module "machine_pool" {
  source = "{{ .Object.Spec.Module.Source }}"
{{- if .Object.Spec.Module.Version }}
  version = "{{ .Object.Spec.Module.Version }}"
{{- end }}
{{- if .Object.Spec.Variables }}
  {{ range $v := .Object.Spec.Variables }}
    {{ $v.Name }} = var.{{ $v.Name }}
  {{- end }}
{{- end }}

  pool_name    = "{{ .Owner.ObjectMeta.Name }}"
  cluster_name = "{{ .Owner.Spec.ClusterName }}" 

  replicas = {{ .Owner.Spec.Replicas }}
}

output "provider_id_list" {
  value = module.machine_pool.provider_id_list
}
`
