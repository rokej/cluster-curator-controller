// Copyright Contributors to the Open Cluster Management project.

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clustercuratorv1 "github.com/open-cluster-management/cluster-curator-controller/pkg/api/v1alpha1"
	"github.com/open-cluster-management/cluster-curator-controller/pkg/controller/launcher"
	"github.com/open-cluster-management/cluster-curator-controller/pkg/jobs/rbac"
	"github.com/open-cluster-management/cluster-curator-controller/pkg/jobs/utils"
)

// ClusterCuratorReconciler reconciles a ClusterCurator object
type ClusterCuratorReconciler struct {
	client.Client
	Kubeset  kubernetes.Interface
	Log      logr.Logger
	Scheme   *runtime.Scheme
	ImageURI string
}

// +kubebuilder:rbac:groups=cluster.open-cluster-management.io.cluster.open-cluster-management.io,resources=clustercurators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.open-cluster-management.io.cluster.open-cluster-management.io,resources=clustercurators/status,verbs=get;update;patch

func (r *ClusterCuratorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("clustercurator", req.NamespacedName)

	var curator clustercuratorv1.ClusterCurator
	if err := r.Get(ctx, req.NamespacedName, &curator); err != nil {
		log.V(2).Info("Resource deleted")
		return ctrl.Result{}, nil
	}

	log.V(3).Info("Reconcile: %v, DesiredCuration: %v, Previous CuratingJob: %v",
		req.NamespacedName, curator.Spec.DesiredCuration, curator.Spec.CuratingJob)

	// Curating work has already started OR no curation work supplied
	if curator.Spec.CuratingJob != "" || curator.Spec.DesiredCuration == "" {
		log.V(3).Info("No curation to do for %v", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Curation flow begins here
	// Apply RBAC required by the curation job
	err := rbac.ApplyRBAC(r.Kubeset, req.Namespace)
	if err := utils.LogError(err); err != nil {
		return ctrl.Result{}, err
	}

	// Launch the curation job
	jobLaunch := launcher.NewLauncher(r.Client, r.Kubeset, r.ImageURI, curator)
	if err := utils.LogError(jobLaunch.CreateJob()); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterCuratorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustercuratorv1.ClusterCurator{}).
		Complete(r)
}