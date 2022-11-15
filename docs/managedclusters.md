# Managed Clusters 

The following CRDs are provided to implement the provisioning of managed clusters (i.e. GKE, EKS, AKS, etc.) using Terraform Cloud.
 
- [TFCManagedControlPlane](#TFCManagedControlPlane) which fulfills the [Cluster](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/cluster.html) and [Control Plane](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/control-plane.html) contracts
- [TFCManagedMachinePool](#TFCManagedMachinePool) which fulfills the [Machine Pool](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/machine-pool.html) contract

## TFCManagedControlPlane

The purpose of the TFManagedControlPlane resource is to configure the Terraform module that will be used to provision the control plane (and any supporting infrastructure such as service accounts) for a managed cluster. 

This resource will fulfil the contracts for both Control Plane and Cluster. 

Example Resource:

```yaml
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: my-cluster
spec:
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: TFCManagedControlPlane
    name: my-cluster
  controlPlaneRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: TFCManagedControlPlane
    name: my-cluster
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: TFCManagedControlPlane
metadata:
  name: my-cluster
spec:
  organization: my-tfc-organization
  workspace: my-controlplane-workspace
  token:
    secretKeyRef:
      name: terraform-cloud-config
      key: token
  version: "1.24"
  module:
    source: my-org/capi/controlplane
    version: 1.0.0
  variables: []
  autoApply: true
```

Example Terraform Module:

See [examples/gke/controlplane](../examples/gke/controlplane).

## TFCManagedMachinePool

The purpose of the TFCManagedMachinePool resource is to configure the Terraform module that will be used to provision a Machine Pool for a managed cluster. 

This resource will fulfil the contract for Machine Pool. 

> :warning: **Machine Pools are still an experimental feature of Cluster API and need to be enabled by setting `EXP_MACHINE_POOL=true` when running `clusterctl init`**

Example:

```yaml
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachinePool
metadata:
  name: my-machine-pool-0
spec:
  clusterName: my-cluster
  replicas: 2
  template:
    spec:
      # NOTE: as this is a managed cluster bootstrap.dataSecretName 
      # should be set to "" to skip bootstrapping
      bootstrap:
        dataSecretName: ""
      clusterName: my-machine
      infrastructureRef:
        name: my-machine-pool-0
        apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
        kind: TFCManagedMachinePool
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: TFCManagedMachinePool
metadata:
  name: my-machine-pool-0
spec: 
  organization: my-tfc-organization
  workspace: my-controlplane-workspace
  token:
    secretKeyRef:
      name: terraform-cloud-config
      key: token
  module:
    source: my-org/capi/machinepool
    version: 1.0.0
  variables: []
  autoApply: true
```

Example Terraform Module:

See [examples/gke/controlplane](../examples/gke/machinepool).