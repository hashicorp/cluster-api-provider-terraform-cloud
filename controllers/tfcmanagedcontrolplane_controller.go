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
	"k8s.io/apimachinery/pkg/types"
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

const tfcManagedControlPlaneFinalizer = "infrastructure.cluster.x-k8s.io/tfc-managed-control-plane"

// TFCManagedControlPlaneReconciler reconciles a TFCManagedControlPlane object
type TFCManagedControlPlaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tfcmanagedcontrolplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tfcmanagedcontrolplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tfcmanagedcontrolplanes/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *TFCManagedControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling TFCManagedControlPlane")

	// get the TFCManagedControlPlane object
	var cluster infrastructurev1alpha1.TFCManagedControlPlane
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("TFCManagedControlPlane has been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Could not locate TFCManagedControlPlane", "name", req.Name)
		return ctrl.Result{}, err
	}

	// Fetch the Cluster object
	ownerCluster, err := util.GetOwnerCluster(ctx, r.Client, cluster.ObjectMeta)
	if err != nil {
		logger.Error(err, "Could not find Cluster object")
		return ctrl.Result{}, nil
	}
	if ownerCluster == nil {
		logger.Info("Cluster object not ready yet")
		return ctrl.Result{}, nil
	}

	// add controller finalizer
	addFinalizer(ctx, r.Client, &cluster, tfcManagedControlPlaneFinalizer)

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
	workspace, err := tfcClient.Workspaces.Read(ctx, cluster.Spec.Organization, cluster.Spec.Workspace)
	if err != nil {
		logger.Error(err, "Error getting Terraform Cloud Workspace")
		return ctrl.Result{}, err
	}

	// run a destroy if the Kubernetes resource is deleted
	// TODO: move this into a reconcileDelete() function
	if !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cluster, tfcManagedControlPlaneFinalizer) {
			logger.Info("Resource is deleted, triggering destroy")

			// trigger destroy run
			_, err := tfcClient.Runs.Create(ctx, tfc.RunCreateOptions{
				Message:   tfc.String(fmt.Sprintf("%s: Destroy Control Plane %q", terraformCloudRunMessage, cluster.ObjectMeta.Name)),
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
				Namespace: cluster.GetNamespace(),
				Name:      fmt.Sprintf("%s-kubeconfig", cluster.GetName()),
			}})
			if err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Error deleting kubeconfig secret")
				return ctrl.Result{}, err
			}

			// TODO: wait until the destroy plan has completed
			controllerutil.RemoveFinalizer(&cluster, tfcManagedControlPlaneFinalizer)
			err = r.Client.Update(ctx, &cluster)
			if err != nil {
				logger.Error(err, "Error removing finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// generate the Terraform config
	terraformConfigPath, configHash, err := terraform.CreateConfiguration(terraform.ManagedClusterConfigurationTemplate, &cluster, &ownerCluster)
	defer os.RemoveAll(terraformConfigPath)
	if err != nil {
		logger.Error(err, "Error generating Terraform configuration")
		return ctrl.Result{}, err
	}

	// upload the terraform configuration
	configurationVersionID := cluster.Status.Terraform.ConfigurationVersionID
	if configurationVersionID == "" || configHash != cluster.Status.Terraform.ConfigurationHash {
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
		cluster.Status.Terraform.ConfigurationVersionID = configurationVersionID
		cluster.Status.Terraform.ConfigurationHash = configHash
		cluster.Status.Terraform.RunID = ""
		cluster.Status.Terraform.RunStatus = ""
		r.Client.Status().Update(ctx, &cluster)
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
	runID := cluster.Status.Terraform.RunID
	if runID == "" {
		// trigger a new run
		logger.Info("Triggering Terraform Cloud Run")
		run, err := tfcClient.Runs.Create(ctx, tfc.RunCreateOptions{
			Message:              tfc.String(fmt.Sprintf("%s: Reconcile Control Plane %q", terraformCloudRunMessage, cluster.ObjectMeta.Name)),
			Workspace:            workspace,
			AutoApply:            tfc.Bool(true),
			ConfigurationVersion: cv,
		})
		if err != nil {
			logger.Error(err, "Error triggering new Terraform run")
			return requeueAfterSeconds(30)
		}

		// set status
		cluster.Status.Terraform.RunID = run.ID
		cluster.Status.Terraform.RunStatus = string(run.Status)
		cluster.Status.Terraform.RunStartedAt = metav1.NewTime(time.Now())
		r.Client.Status().Update(ctx, &cluster)
		return ctrl.Result{Requeue: true, RequeueAfter: 60 * time.Second}, nil
	}

	run, err := tfcClient.Runs.Read(ctx, runID)
	if err != nil {
		logger.Error(err, "Error reading Terraform Cloud Run")
		return requeueAfterSeconds(30)
	}

	cluster.Status.Terraform.RunStatus = string(run.Status)
	r.Client.Status().Update(ctx, &cluster)

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
		// that the cluster is ready after a terraform apply is done
		cluster.Status.Initialized = true
		cluster.Status.Ready = true
		cluster.Status.Terraform.RunFinishedAt = metav1.NewTime(time.Now())
		r.Client.Status().Update(ctx, &cluster)

		outputs, err := tfcClient.StateVersions.ListOutputs(ctx,
			workspace.CurrentStateVersion.ID, &tfc.StateVersionOutputsListOptions{})
		if err != nil {
			logger.Error(err, "Error reading terraform run state")
			return requeueAfterSeconds(30)
		}
		// TODO: getOutput() function
		var kubeconfig string
		for _, o := range outputs.Items {
			switch o.Name {
			case "control_plane_endpoint_host":
				cluster.Spec.ControlPlaneEndpoint.Host = o.Value.(string)
			case "control_plane_endpoint_port":
				cluster.Spec.ControlPlaneEndpoint.Port = int32(o.Value.(float64))
			case "kubeconfig":
				kubeconfig = o.Value.(string)
			}
		}
		r.Client.Update(ctx, &cluster)

		// create secret containing kubeconfig
		// TODO: createKubeconfig function
		secret := corev1.Secret{
			TypeMeta: v1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace: cluster.GetNamespace(),
				Name:      fmt.Sprintf("%s-kubeconfig", cluster.GetName()),
			},
			StringData: map[string]string{
				"value": kubeconfig,
			},
		}
		err = r.Client.Patch(ctx, &secret, client.Apply, client.FieldOwner("terraform-cloud-cluster"))
		if err != nil {
			logger.Error(err, "Error creating kubeconfig Secret")
			return ctrl.Result{}, err
		}
	default:
		// run is still in progress
		return requeueAfterSeconds(30)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TFCManagedControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.TFCManagedControlPlane{}).
		Complete(r)
}
