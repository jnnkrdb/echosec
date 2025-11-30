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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
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
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clusterobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clusterobjects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.jnnkrdb.de,resources=clusterobjects/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
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
	var co = &clusterv1alpha1.ClusterObject{}
	if err := r.Get(ctx, req.NamespacedName, co, &client.GetOptions{}); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			_log.Error(err, "error fetching object from cluster")
		}
		return ctrl.Result{}, err
	}

	// request a list of namespaces, to parse through the list and
	// then check every namespace with the give item
	var namespaces = &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces, &client.ListOptions{}); err != nil {

		_log.Error(err, "error fetching list of namespaces from cluster")

		r.Recorder.Eventf(co, "Warning", "ListNamespacesError", "error listing namespaces: %v", err)

		if e := co.SetCondition(
			ctx, r.Client, clusterv1alpha1.Condition_Ready, metav1.ConditionFalse,
			"FailedToListNamespaces", "error listing namespaces: %v", err,
		); e != nil {
			return ctrl.Result{}, e
		}

		return ctrl.Result{}, err
	}

	for _, namespace := range namespaces.Items {

		// reconcile the object for a specific namespace, if an error occurs, then throw reconcile error
		if err := r.ReconcileObjectForNamespace(ctx, co, namespace); err != nil {
			return ctrl.Result{}, err
		}
	}

	_log.Info("reconciled")

	r.Recorder.Eventf(co, "Normal", "ReconciledObject", "successfully cloned resource in required namespaces")

	return ctrl.Result{}, co.SetCondition(ctx, r.Client, clusterv1alpha1.Condition_Ready,
		metav1.ConditionTrue, "DeployedResource",
		"successfully deployed resource [%s/%s:%s]",
		co.Resource.GetAPIVersion(),
		co.Resource.GetKind(),
		co.Resource.GetName(),
	)
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

// ------------------------------------------------------ status functions

/*
this function checks the requested resource, wether it should exist in a namespace, or not.

following cases should be considered:
 1. secret should not exist and does not exist -> ignore
 2. secret should exist but does not -> create
 3. secret should exist and it exists -> update
 4. secret should not exist but does exist -> delete
*/
func (r *ClusterObjectReconciler) ReconcileObjectForNamespace(ctx context.Context, co *clusterv1alpha1.ClusterObject, namespace corev1.Namespace) error {
	var _log = log.FromContext(ctx).WithValues(
		"apiVersion", co.Resource.GetAPIVersion(),
		"kind", co.Resource.GetKind(),
		"name", co.Resource.GetName(),
		"namespace", namespace.GetName(),
	)

	_log.V(3).Info("check object")

	//var requestedObject = types.NamespacedName{Namespace: namespace.Name, Name: co.Resource.GetName()}
	var typedObject = co.Resource.DeepCopy()
	var shouldExist, doesExist bool

	// check, if the object should exist in the namespace
	if se, err := co.RegexRules.ShouldExistInNamespace(namespace.Name); err != nil {

		_log.Error(err, "error calculating wether the item should exist or not")

		r.Recorder.Eventf(co, "Warning", "CalculatingNamespaceError",
			"error calculating wether the item should exist or not: %v", err)

		if e := co.SetCondition(ctx, r.Client, clusterv1alpha1.Condition_Ready, metav1.ConditionFalse,
			"FailedToCalculateNamespace", "error calculating wether the item should exist or not: %v", err); e != nil {
			return e
		}

		return err

	} else {

		shouldExist = se
	}

	// check, if the requested object does exist in the namespace
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace.Name, Name: co.Resource.GetName()},
		typedObject, &client.GetOptions{}); err != nil {

		if client.IgnoreNotFound(err) != nil {

			_log.Error(err, "error receiving the object from the cluster")

			r.Recorder.Eventf(co, "Warning", "ReceivingClusterObjectError",
				"error receiving the object from the cluster: %v", err)

			if e := co.SetCondition(ctx, r.Client, clusterv1alpha1.Condition_Ready, metav1.ConditionFalse,
				"FailedToReceiveObjectFromCluster", "error receiving the object from the cluster: %v", err); e != nil {
				return e
			}

			return err
		}

		doesExist = false

	} else {

		doesExist = true
	}

	_log.V(3).Info("state calculated", "shouldExist", shouldExist, "doesExist", doesExist)

	// update the values of the tempObject (only really needed for creating or updating)
	typedObject = co.Resource.DeepCopy()
	typedObject.SetNamespace(namespace.Name)

	// set the owners reference
	// this is required for watching the dependent objects
	if err := controllerutil.SetControllerReference(co, typedObject, r.Scheme); err != nil {

		_log.Error(err, "unable to set owners reference")

		r.Recorder.Eventf(co, "Warning", "SettingOwnerReferenceError",
			"unable to set owners reference: %v", err)

		if e := co.SetCondition(ctx, r.Client, clusterv1alpha1.Condition_Ready, metav1.ConditionFalse,
			"FailedToSetOwnerReference", "unable to set owners reference: %v", err); e != nil {
			return e
		}
		return err
	}

	// after calculating the current state, handle the 4 cases
	switch {
	case !shouldExist && !doesExist: // --------------------------------------------------------- case 1 -> ignore
		_log.V(3).Info("ignoring")

	case shouldExist && !doesExist: // --------------------------------------------------------- case 2 -> create
		_log.V(3).Info("creating")
		if err := r.Create(ctx, typedObject, &client.CreateOptions{}); err != nil {
			_log.Error(err, "error creating object")

			r.Recorder.Eventf(co, "Warning", "CreatingObjectError", "error creating object: %v", err)

			if e := co.SetCondition(ctx, r.Client, clusterv1alpha1.Condition_Ready, metav1.ConditionFalse,
				"FailedCreatingObject", "error creating object: %v", err); e != nil {
				return e
			}
			return err
		}

	case shouldExist && doesExist: // --------------------------------------------------------- case 3 -> update
		_log.V(3).Info("updating")
		if err := r.Update(ctx, typedObject, &client.UpdateOptions{}); err != nil {
			_log.Error(err, "error updating object")

			r.Recorder.Eventf(co, "Warning", "UpdatingObjectError", "error updating object: %v", err)

			if e := co.SetCondition(ctx, r.Client, clusterv1alpha1.Condition_Ready, metav1.ConditionFalse,
				"FailedUpdatingObject", "error updating object: %v", err); e != nil {
				return e
			}
			return err
		}

	case !shouldExist && doesExist: // --------------------------------------------------------- case 4 -> delete
		_log.V(3).Info("deleting")
		if err := r.Delete(ctx, typedObject, &client.DeleteOptions{}); client.IgnoreNotFound(err) != nil {
			_log.Error(err, "error deleting object")

			r.Recorder.Eventf(co, "Warning", "DeletingObjectError", "error deleting object: %v", err)

			if e := co.SetCondition(ctx, r.Client, clusterv1alpha1.Condition_Ready, metav1.ConditionFalse,
				"FailedDeletingObject", "error deleting object: %v", err); e != nil {
				return e
			}
			return err
		}
	}

	return nil
}
