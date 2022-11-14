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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TFCManagedMachinePoolSpec defines the desired state of TFCManagedMachinePool
type TFCManagedMachinePoolSpec struct {
	// Organization is the name of the Terraform Cloud organization to use
	Organization string `json:"organization"`

	// Workspace is the name of the Terraform Cloud Workspace to execute the terraform run in
	// TODO: change this to a struct that supports ID or name
	Workspace string `json:"workspace"`

	// Token is the API token for accessing Terraform Cloud
	Token Token `json:"token"`

	// Module is the Terraform module to use for provisioning the Kubernetes Cluster
	Module TerraformModule `json:"module"`

	// AutoApply configures if plans should be applied straight away or manually approved in the Terraform Cloud UI
	AutoApply bool `json:"autoApply"`

	// Variables is the list of variables to supply to the Terraform module which creates the Kubernetes Cluster
	Variables []Variable `json:"variables"`

	// ProviderIDList is a list of cloud provider IDs identifying the instances.
	ProviderIDList []string `json:"providerIDList,omitempty"`
}

// TFCManagedMachinePoolStatus defines the observed state of TFCManagedMachinePool
type TFCManagedMachinePoolStatus struct {
	Ready     bool            `json:"ready,omitempty"`
	Terraform TerraformStatus `json:"terraform,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Organization",type=string,JSONPath=`.spec.workspace`
//+kubebuilder:printcolumn:name="Workspace",type=string,JSONPath=`.spec.organization`
//+kubebuilder:printcolumn:name="Module",type=string,JSONPath=`.spec.module.source`
//+kubebuilder:printcolumn:name="Module Version",type=string,JSONPath=`.spec.module.version`
//+kubebuilder:printcolumn:name="Run Status",type=string,JSONPath=`.status.terraform.runStatus`

// TFCManagedMachinePool is the Schema for the tfcmanagedmachinepools API
type TFCManagedMachinePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TFCManagedMachinePoolSpec   `json:"spec,omitempty"`
	Status TFCManagedMachinePoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TFCManagedMachinePoolList contains a list of TFCManagedMachinePool
type TFCManagedMachinePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TFCManagedMachinePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TFCManagedMachinePool{}, &TFCManagedMachinePoolList{})
}
