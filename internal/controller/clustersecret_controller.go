/*
Copyright 2025.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	clusterv1alpha1 "github.com/jnnkrdb/echosec/api/v1alpha1"
)

// ClusterSecretReconciler reconciles a ClusterSecret object
type ClusterSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clustersecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clustersecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clustersecrets/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets/status,verbs=get;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *ClusterSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var _log = log.FromContext(ctx).WithValues("req.Name", req.Name, "req.Namespac+e", req.Namespace)

	// receive the object, which should be reconciled
	var recObj = &clusterv1alpha1.ClusterSecret{}
	if err := r.Get(ctx, req.NamespacedName, recObj, &client.GetOptions{}); err != nil {
		_log.Error(err, "error fetching object from cluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	_log.Info("clustersecret handled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.ClusterSecret{}).
		Named("clustersecret").
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
