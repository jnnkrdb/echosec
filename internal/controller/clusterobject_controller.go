/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clusterv1alpha1 "github.com/jnnkrdb/echosec/api/v1alpha1"
)

// ClusterObjectReconciler reconciles a ClusterObject object
type ClusterObjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clusterobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clusterobjects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clusterobjects/finalizers,verbs=update

// +kubebuilder:rbac:groups=*,resources=*,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterObject object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *ClusterObjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var _log = log.FromContext(ctx)

	// -------------------------------------------------------- meta handling
	// receive the object, which should be reconciled
	var recObj = &clusterv1alpha1.ClusterObject{}
	if err := r.Get(ctx, req.NamespacedName, recObj, &client.GetOptions{}); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			_log.Error(err, "error fetching object from cluster")
		}
		return ctrl.Result{}, err
	}

	// -------------------------------------------------------- item handling

	var errorsList []error

	// handle the cluster secret
	var namespaces = &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces, &client.ListOptions{}); err != nil {
		_log.Error(err, "error fetching list of namespaces from cluster")
		return ctrl.Result{}, err
	}

	for _, namespace := range namespaces.Items {
		var requestedObject = types.NamespacedName{Namespace: namespace.Name, Name: recObj.Resource.GetName()}
		var sourceObject = recObj.Resource.DeepCopy()
		var createObject = recObj.Resource.DeepCopy()

		nsLog := _log.V(3).WithValues("requested-object", requestedObject, "requested-object-kind", sourceObject.GetKind())
		nsLog.Info("check item in namespace")

		// ignore namespace if marked for deletion
		if !namespace.DeletionTimestamp.IsZero() {
			nsLog.Info("namespace marked for deletion -> ignore")
			continue
		}

		/*
			following cases should be considered:
			1. secret should not exist and does not exist -> ignore
			2. secret should exist but does not -> create
			3. secret should exist and it exists -> update
			4. secret should not exist but does exist -> delete

		*/

		var shouldExist, doesExist bool
		// should the item exist ?
		if se, err := recObj.RegexRules.ShouldExistInNamespace(requestedObject.Namespace); err != nil {
			nsLog.Error(err, "error calculating wether the item should exist or not")
			errorsList = append(errorsList, err)
			continue
		} else {
			shouldExist = se
		}

		// does the item exist ?
		if err := r.Get(ctx, requestedObject, sourceObject, &client.GetOptions{}); err != nil {
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

		// update the values of the tempObject (only really needed for creating or updating)
		sourceObject = createObject

		// set the owners reference
		// this is required for watching the dependent objects
		if err := controllerutil.SetControllerReference(recObj, sourceObject, r.Scheme); err != nil {
			nsLog.Error(err, "unable to set owners reference, stopping reconciliation")
			return ctrl.Result{}, err
		}

		// after calculating the current state, handle the 4 cases
		switch {
		case !shouldExist && !doesExist: // --------------------------------------------------------- case 1 -> ignore
			nsLog.Info("the requested object does not exist and should not exist -> ignoring")
			continue

		case shouldExist && !doesExist: // --------------------------------------------------------- case 2 -> create
			nsLog.Info("the requested object does not exist but should exist -> creating")
			if err := r.Create(ctx, sourceObject, &client.CreateOptions{}); err != nil {
				nsLog.Error(err, "error creating object")
				errorsList = append(errorsList, err)
				continue
			}

		case shouldExist && doesExist: // --------------------------------------------------------- case 3 -> update
			nsLog.Info("the requested object does exist and should exist -> updating")
			if err := r.Update(ctx, sourceObject, &client.UpdateOptions{}); err != nil {
				nsLog.Error(err, "error updating object")
				errorsList = append(errorsList, err)
				continue
			}

		case !shouldExist && doesExist: // --------------------------------------------------------- case 4 -> delete
			nsLog.Info("the requested object does exist but should not exist -> deleting")
			if err := r.Delete(ctx, sourceObject, &client.DeleteOptions{}); client.IgnoreNotFound(err) != nil {
				nsLog.Error(err, "error deleting object from cluster")
				errorsList = append(errorsList, err)
				continue
			}
		}
	}

	// -------------------------------------------------------- finish

	if len(errorsList) > 0 {
		e := fmt.Errorf("errorsList not empty")
		_log.Error(e, "clusterobject handled, but with errors", "errors", errorsList)
		return ctrl.Result{}, e
	}

	_log.Info("clusterobject handled without errors ")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterObjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.ClusterObject{}).
		Named("clusterobject").
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.ResourceVersionChangedPredicate{},
			),
		).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(
				func(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
					var _log = log.FromContext(ctx)
					// trigger reconciliation for all clusterobjects
					var list = &clusterv1alpha1.ClusterObjectList{}
					if err := mgr.GetClient().List(ctx, list, &client.ListOptions{}); err != nil {
						_log.Error(err, "error receiving list of clusterobjects, cannot invoke reconciliation")
						return
					}
					for _, co := range list.Items {
						requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: co.Name}})
					}
					return
				},
			),
		).
		Complete(r)
}
