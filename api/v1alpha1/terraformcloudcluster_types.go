/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TerraformModule defines the source and version for a Terraform module
type TerraformModule struct {
	// Source is the Terraform Registry or HTTP URL of the module source
	Source string `json:"source"`

	// Version is the semantic version of the Terraform Module
	Version string `json:"version"`
}

// Variable is a Terraform Variable
type Variable struct {
	// Name is the name of the variable
	Name string `json:"name"`

	// TODO: sensitive
	// TODO: allow setting the value
}

// Token refers to a Kubernetes Secret object within the same namespace as the Workspace object
type Token struct {
	// Selects a key of a secret in the workspace's namespace
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef"`
}

// TerraformCloudClusterSpec defines the desired state of TerraformCloudCluster
type TerraformCloudClusterSpec struct {
	// Organization is the name of the Terraform Cloud organization to use
	Organization string `json:"organization"`

	// Workspace is the name of the Terraform Cloud Workspace to execute the terraform run in
	// TODO: change this to a struct that supports ID or name
	Workspace string `json:"workspace"`

	// Token is the API token for accessing Terraform Cloud
	Token Token `json:"token"`

	// Module is the Terraform module to use for provisioning the Kubernetes Cluster
	Module TerraformModule `json:"module"`

	// Version is the Kubernetes cluster version to provision
	Version string `json:"version"`

	// AutoApply configures if plans should be applied straight away or manually approved in the Terraform Cloud UI
	AutoApply bool `json:"autoApply"`

	// Variables is the list of variables to supply to the Terraform module which creates the Kubernetes Cluster
	Variables []Variable `json:"variables"`

	// ControlPlaneEndpoint is the endpoint for the control plane
	ControlPlaneEndpoint clusterv1beta1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
}

// TerraformCloudClusterStatus defines the observed state of TerraformCloudCluster
type TerraformCloudClusterStatus struct {
	// +kubebuilder:default=false
	Ready       bool            `json:"ready"`
	Initialized bool            `json:"initialized"`
	Terraform   TerraformStatus `json:"terraform,omitempty"`
}

// TerraformStatus defines status information about the terraform workspace
type TerraformStatus struct {
	// subresource for TerraformRun
	RunID                  string      `json:"runID,omitempty"`
	RunStatus              string      `json:"runStatus,omitempty"`
	RunStartedAt           metav1.Time `json:"runStartedAt,omitempty"`
	RunFinishedAt          metav1.Time `json:"runFinishedAt,omitempty"`
	ConfigurationVersionID string      `json:"configurationVersionID,omitempty"`
	ConfigurationHash      string      `json:"configurationHash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
//+kubebuilder:printcolumn:name="Organization",type=string,JSONPath=`.spec.workspace`
//+kubebuilder:printcolumn:name="Workspace",type=string,JSONPath=`.spec.organization`
//+kubebuilder:printcolumn:name="Run Status",type=string,JSONPath=`.status.terraform.runStatus`

// TerraformCloudCluster is the Schema for the terraformcloudclusters API
type TerraformCloudCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TerraformCloudClusterSpec   `json:"spec,omitempty"`
	Status TerraformCloudClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TerraformCloudClusterList contains a list of TerraformCloudCluster
type TerraformCloudClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TerraformCloudCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TerraformCloudCluster{}, &TerraformCloudClusterList{})
}
