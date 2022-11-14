<a href="https://cloud.hashicorp.com/products/terraform">
    <img src=".github/tf_logo.png" alt="Terraform logo" title="Terraform Cloud" align="left" height="50" />
</a>

# Kubernetes Cluster API Provider for Terraform Cloud

> **Warning**
> Please note that this is a technical preview and is for experimental purposes only 

Kubernetes-native declarative infrastructure using Terraform Cloud.

## What is the Cluster API Provider for Terraform Cloud?

The [Cluster API](https://github.com/kubernetes-sigs/cluster-apiterra) project brings declarative Kubernetes-style APIs to cluster creation, configuration and management. This provider allows you to create [Terraform Modules](https://developer.hashicorp.com/terraform/language/modules) to implement Cluster API's contracts and run them in [Terraform Cloud](https://cloud.hashicorp.com/products/terraform) to provision the infrastructure using Kubernetes as your source of truth.  

## Supported Cluster API contracts

The provider currently implements the contracts that allow the infrastructure for managed clusters and machine pools to be provisioned. 

- [Cluster](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/cluster.html) and [Control Plane](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/control-plane.html) are fulfilled by TFCManagedControlPlane  
- [MachinePool](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/machine-pool.html) is fulfilled by TFCManagedMachinePool 


## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster

1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/cluster-api-provider-terraform-cloud:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/cluster-api-provider-terraform-cloud:tag
```

### Uninstall CRDs

To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing

// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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

