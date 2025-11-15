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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	clusterv1alpha1 "github.com/jnnkrdb/echosec/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// ClusterSecretReconciler reconciles a ClusterSecret object
type ClusterSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clustersecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clustersecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clustersecrets/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

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
	var _log = log.FromContext(ctx)

	// -------------------------------------------------------- meta handling
	// receive the object, which should be reconciled
	var recObj = &clusterv1alpha1.ClusterSecret{}
	if err := r.Get(ctx, req.NamespacedName, recObj, &client.GetOptions{}); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			_log.Error(err, "error fetching object from cluster")
		}
		return ctrl.Result{}, err
	}

	// handle finalization if required
	/*
		if deleted, err := recObj.Finalize(log.IntoContext(ctx, _log.V(3)), r.Client); err != nil {
			_log.Error(err, "error handling deletion request")
			return ctrl.Result{}, err

		} else if deleted {

			_log.Info("clustersecret deleted from cluster")
			return ctrl.Result{}, nil
		}
	*/

	// -------------------------------------------------------- item handling

	var errorsList []error

	// handle the cluster secret
	var namespaces = &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces, &client.ListOptions{}); err != nil {
		_log.Error(err, "error fetching list of namespaces from cluster")
		return ctrl.Result{}, err
	}

	for _, namespace := range namespaces.Items {
		var requestedSecret = types.NamespacedName{Namespace: namespace.Name, Name: recObj.GetDependentsName()}
		var tempSecret = &corev1.Secret{}

		nsLog := _log.V(3).WithValues("requested-secret", requestedSecret)
		nsLog.Info("check item in namespace")

		/*
			following cases should be considered:
			1. secret should not exist and does not exist -> ignore
			2. secret should exist but does not -> create
			3. secret should exist and it exists -> update
			4. secret should not exist but does exist -> delete

		*/

		var shouldExist, doesExist bool
		// should the item exist ?
		if se, err := recObj.Spec.RegexRules.ShouldExistInNamespace(requestedSecret.Namespace); err != nil {
			nsLog.Error(err, "error calculating wether the item should exist or not")
			errorsList = append(errorsList, err)
			continue
		} else {
			shouldExist = se
		}

		// does the item exist ?
		if err := r.Get(ctx, requestedSecret, tempSecret, &client.GetOptions{}); err != nil {
			if client.IgnoreNotFound(err) != nil {
				nsLog.Error(err, "error fetching object from cluster")
				errorsList = append(errorsList, err)
				continue
			}
			doesExist = false
		} else {
			doesExist = true
		}

		// update log with new values
		nsLog = nsLog.WithValues("shouldExist", shouldExist, "doesExist", doesExist)

		// update the values of the tempSecret (only really needed for creating or updating)
		tempSecret.Name = requestedSecret.Name
		tempSecret.Namespace = requestedSecret.Namespace
		tempSecret.Data = recObj.Spec.Data
		tempSecret.StringData = recObj.Spec.StringData
		tempSecret.Type = recObj.Spec.Type

		// set the owners reference
		// this is required for watching the dependent objects
		if err := controllerutil.SetControllerReference(recObj, tempSecret, r.Scheme); err != nil {
			nsLog.Error(err, "unable to set owners reference, stopping reconciliation")
			return ctrl.Result{}, err
		}

		// after calculating the current state, handle the 4 cases
		switch {
		case !shouldExist && !doesExist: // --------------------------------------------------------- case 1 -> ignore
			nsLog.Info("the requested secret does not exist and should not exist -> ignoring")
			continue

		case shouldExist && !doesExist: // --------------------------------------------------------- case 2 -> create
			nsLog.Info("the requested secret does not exist but should exist -> creating")
			if err := r.Create(ctx, tempSecret, &client.CreateOptions{}); err != nil {
				nsLog.Error(err, "error creating object")
				errorsList = append(errorsList, err)
				continue
			}

		case shouldExist && doesExist: // --------------------------------------------------------- case 3 -> update
			nsLog.Info("the requested secret does exist and should exist -> updating")
			if err := r.Update(ctx, tempSecret, &client.UpdateOptions{}); err != nil {
				nsLog.Error(err, "error updating object")
				errorsList = append(errorsList, err)
				continue
			}

		case !shouldExist && doesExist: // --------------------------------------------------------- case 4 -> delete
			nsLog.Info("the requested secret does exist but should not exist -> deleting")
			if err := r.Delete(ctx, tempSecret, &client.DeleteOptions{}); client.IgnoreNotFound(err) != nil {
				nsLog.Error(err, "error deleting object from cluster")
				errorsList = append(errorsList, err)
				continue
			}
		}
	}

	// -------------------------------------------------------- finish

	if len(errorsList) > 0 {
		e := fmt.Errorf("errorsList not empty")
		_log.Error(e, "clustersecret handled, but with errors", "errors", errorsList)
		return ctrl.Result{}, e
	}

	_log.Info("clustersecret handled without errors ")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.ClusterSecret{}).
		Named("clustersecret").
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
