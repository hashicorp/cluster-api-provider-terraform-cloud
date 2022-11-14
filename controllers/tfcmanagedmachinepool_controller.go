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

package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	expclusterv1beta1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrastructurev1alpha1 "github.com/hashicorp/cluster-api-provider-terraform-cloud/api/v1alpha1"
	"github.com/hashicorp/cluster-api-provider-terraform-cloud/terraform"
	"github.com/hashicorp/go-tfe"
	tfc "github.com/hashicorp/go-tfe"
)

const tfcManagedMachinePoolFinalizer = "infrastructure.cluster.x-k8s.io/tfc-managed-machine-pool"

// TFCManagedMachinePoolReconciler reconciles a TFCManagedMachinePool object
type TFCManagedMachinePoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tfcmanagedmachinepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tfcmanagedmachinepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tfcmanagedmachinepools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TFCManagedMachinePool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *TFCManagedMachinePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling TFCManagedMachinePool")

	// get the TFCManagedControlPlane object
	var machinePool infrastructurev1alpha1.TFCManagedMachinePool
	if err := r.Get(ctx, req.NamespacedName, &machinePool); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("TFCManagedMachinePool has been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Could not locate TFCManagedControlPlane", "name", req.Name)
		return ctrl.Result{}, err
	}

	// get the owner MachinePool object
	ownerMachinePool, err := getOwnerMachinePool(ctx, r.Client, machinePool.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if ownerMachinePool == nil {
		logger.Info("MachinePool object not ready yet")
		return ctrl.Result{}, nil
	}

	// get the owner Cluster object
	ownerCluster, err := util.GetClusterFromMetadata(ctx, r.Client, machinePool.ObjectMeta)
	if err != nil {
		logger.Error(err, "Could not find owner Cluster object")
		return ctrl.Result{}, nil
	}
	if ownerCluster == nil || !ownerCluster.Status.ControlPlaneReady {
		// TODO: perhaps we should make waiting for this this configurable
		logger.Info("Control plane is not ready yet")
		return requeueAfterSeconds(10)
	}

	// add controller finalizer
	addFinalizer(ctx, r.Client, &machinePool, tfcManagedMachinePoolFinalizer)

	// read the token secret
	var tokenSecret corev1.Secret
	err = r.Client.Get(ctx, types.NamespacedName{Name: terraformCloudTokenSecretName, Namespace: req.Namespace}, &tokenSecret)
	if err != nil {
		logger.Error(err, "Could not find token Secret object")
		return ctrl.Result{}, err
	}
	token := string(tokenSecret.Data["value"])

	// create the TFC client
	tfcClient, err := tfc.NewClient(&tfc.Config{
		Token: token,
	})
	if err != nil {
		logger.Error(err, "Error creating Terraform Cloud client")
		return ctrl.Result{}, err
	}

	// get the TFC workspace
	workspace, err := tfcClient.Workspaces.Read(ctx, machinePool.Spec.Organization, machinePool.Spec.Workspace)
	if err != nil {
		logger.Error(err, "Error getting Terraform Cloud Workspace")
		return ctrl.Result{}, err
	}

	// run a destroy if the Kubernetes resource is deleted
	// TODO: move this into a reconcileDelete() function
	if !machinePool.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&machinePool, tfcManagedMachinePoolFinalizer) {
			logger.Info("Resource is deleted, triggering destroy")

			// trigger destroy run
			_, err := tfcClient.Runs.Create(ctx, tfc.RunCreateOptions{
				Message:   tfc.String(fmt.Sprintf("%s: Destroy MachinePool %q", terraformCloudRunMessage, machinePool.ObjectMeta.Name)),
				Workspace: workspace,
				AutoApply: tfc.Bool(true),
				IsDestroy: tfc.Bool(true),
			})
			if err != nil {
				logger.Error(err, "Error triggering destroy run")
				return ctrl.Result{}, err
			}

			// delete the kubeconfig secret
			err = r.Client.Delete(ctx, &corev1.Secret{ObjectMeta: v1.ObjectMeta{
				Namespace: machinePool.GetNamespace(),
				Name:      fmt.Sprintf("%s-kubeconfig", machinePool.GetName()),
			}})
			if err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Error deleting kubeconfig secret")
				return ctrl.Result{}, err
			}

			// TODO: wait until the destroy plan has completed
			controllerutil.RemoveFinalizer(&machinePool, tfcManagedMachinePoolFinalizer)
			err = r.Client.Update(ctx, &machinePool)
			if err != nil {
				logger.Error(err, "Error removing finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// generate the Terraform config
	terraformConfigPath, configHash, err := terraform.CreateConfiguration(terraform.ManagedMachinePoolConfigurationTemplate, &machinePool, &ownerMachinePool)
	defer os.RemoveAll(terraformConfigPath)
	if err != nil {
		logger.Error(err, "Error generating Terraform configuration")
		return ctrl.Result{}, err
	}

	// upload the terraform configuration
	configurationVersionID := machinePool.Status.Terraform.ConfigurationVersionID
	if configurationVersionID == "" || configHash != machinePool.Status.Terraform.ConfigurationHash {
		// create a new ConfigurationVersion
		logger.Info("Creating new Terraform ConfigurationVersion")
		cv, err := tfcClient.ConfigurationVersions.Create(ctx, workspace.ID, tfc.ConfigurationVersionCreateOptions{
			AutoQueueRuns: tfc.Bool(false),
		})
		if err != nil {
			logger.Error(err, "Error creating new Terraform Cloud ConfigurationVersion")
			return requeueAfterSeconds(30)
		}

		// upload the terraform config
		logger.Info("Uploading Terraform Configuration")
		err = tfcClient.ConfigurationVersions.Upload(ctx, cv.UploadURL, terraformConfigPath)
		if err != nil {
			logger.Error(err, "Error uploading configuration to ConfigurationVersion")
			return requeueAfterSeconds(30)
		}

		configurationVersionID = cv.ID
		machinePool.Status.Terraform.ConfigurationVersionID = configurationVersionID
		machinePool.Status.Terraform.ConfigurationHash = configHash
		machinePool.Status.Terraform.RunID = ""
		machinePool.Status.Terraform.RunStatus = ""
		r.Client.Status().Update(ctx, &machinePool)
		return requeueAfterSeconds(30)
	}

	cv, err := tfcClient.ConfigurationVersions.Read(ctx, configurationVersionID)
	if err != nil {
		logger.Error(err, "Error reading ConfigurationVersion")
		return requeueAfterSeconds(30)
	}

	if cv.Status != tfe.ConfigurationUploaded {
		logger.Info("ConfigurationVersion not ready yet")
		return ctrl.Result{Requeue: true, RequeueAfter: 60 * time.Second}, nil
	}

	// check if there is a run in progress
	runID := machinePool.Status.Terraform.RunID
	if runID == "" {
		// trigger a new run
		logger.Info("Triggering Terraform Cloud Run")
		run, err := tfcClient.Runs.Create(ctx, tfc.RunCreateOptions{
			Message:              tfc.String(fmt.Sprintf("%s: Reconcile MachinePool %q", terraformCloudRunMessage, machinePool.ObjectMeta.Name)),
			Workspace:            workspace,
			AutoApply:            tfc.Bool(true),
			ConfigurationVersion: cv,
		})
		if err != nil {
			logger.Error(err, "Error triggering new Terraform run")
			return requeueAfterSeconds(30)
		}

		// set status
		machinePool.Status.Terraform.RunID = run.ID
		machinePool.Status.Terraform.RunStatus = string(run.Status)
		machinePool.Status.Terraform.RunStartedAt = metav1.NewTime(time.Now())
		r.Client.Status().Update(ctx, &machinePool)
		return ctrl.Result{Requeue: true, RequeueAfter: 60 * time.Second}, nil
	}

	run, err := tfcClient.Runs.Read(ctx, runID)
	if err != nil {
		logger.Error(err, "Error reading Terraform Cloud Run")
		return requeueAfterSeconds(30)
	}

	machinePool.Status.Terraform.RunStatus = string(run.Status)
	r.Client.Status().Update(ctx, &machinePool)

	switch run.Status {
	case tfc.RunDiscarded:
		logger.Info("The Terraform Cloud run was discarded")
	case tfc.RunCanceled:
		logger.Info("The Terraform Cloud run was cancelled")
	case tfc.RunErrored:
		logger.Info("The Terraform Cloud run produced an error")
	case tfc.RunPlannedAndFinished:
	case tfc.RunApplied:
		// TODO: we're going to want some kind of standard way to confirm
		// that the machine pool is ready after a terraform apply is done
		machinePool.Status.Ready = true
		machinePool.Status.Terraform.RunFinishedAt = metav1.NewTime(time.Now())
		r.Client.Status().Update(ctx, &machinePool)

		outputs, err := tfcClient.StateVersions.ListOutputs(ctx,
			workspace.CurrentStateVersion.ID, &tfc.StateVersionOutputsListOptions{})
		if err != nil {
			logger.Error(err, "Error reading terraform run state")
			return requeueAfterSeconds(30)
		}
		// TODO: getOutput() function
		for _, o := range outputs.Items {
			switch o.Name {
			case "provider_id_list":
				providerIDList := []string{}
				for _, v := range o.Value.([]interface{}) {
					providerIDList = append(providerIDList, v.(string))
				}
				machinePool.Spec.ProviderIDList = providerIDList
			}
		}
		r.Client.Update(ctx, &machinePool)
	default:
		// run is still in progress
		return requeueAfterSeconds(30)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TFCManagedMachinePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.TFCManagedMachinePool{}).
		Complete(r)
}

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func getOwnerMachinePool(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*expclusterv1beta1.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind != "MachinePool" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, err
		}
		if gv.Group == expclusterv1beta1.GroupVersion.Group {
			return getMachinePoolByName(ctx, c, obj.Namespace, ref.Name)
		}
	}
	return nil, nil
}

// getMachinePoolByName finds and return a Machine object using the specified params.
func getMachinePoolByName(ctx context.Context, c client.Client, namespace, name string) (*expclusterv1beta1.MachinePool, error) {
	m := &expclusterv1beta1.MachinePool{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := c.Get(ctx, key, m); err != nil {
		return nil, err
	}
	return m, nil
}
