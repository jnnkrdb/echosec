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

	clusterv1alpha1 "github.com/jnnkrdb/r8r/api/v1alpha1"
)

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
					for _, clusterObject := range list.Items {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name: clusterObject.Name,
							},
						})
					}
					return
				},
			),
		).
		Complete(r)
}

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

	var clusterObject = &clusterv1alpha1.ClusterObject{}

	if err := r.Get(ctx, req.NamespacedName, clusterObject, &client.GetOptions{}); err != nil {
		if err = client.IgnoreNotFound(err); err != nil {
			_log.Error(err, "error fetching object from cluster")
		}
		return ctrl.Result{}, err
	}

	_log.V(5).Info("clusterobject content", "*clusterObject", *clusterObject)

	// request a list of namespaces, to parse through the list and
	// then check every namespace with the give item
	var namespaces = &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces, &client.ListOptions{}); err != nil {
		return ctrl.Result{}, r.throwOnError(
			ctx,
			clusterObject,
			err,
			"NamespaceGathering",
			"error fetching list of namespaces from cluster")
	}

	// request a list of namespaces, which are required to inherit the defined object
	labelselector, err := metav1.LabelSelectorAsSelector(clusterObject.Replicator.LabelSelector)
	if err != nil {
		return ctrl.Result{}, r.throwOnError(
			ctx,
			clusterObject,
			err,
			"LabelSelectorFetching",
			"error fetching labelselector from clusterobject")
	}
	var requiredNamespaces = &corev1.NamespaceList{}
	if err := r.List(ctx, requiredNamespaces, &client.ListOptions{LabelSelector: labelselector}); err != nil {
		return ctrl.Result{}, r.throwOnError(
			ctx,
			clusterObject,
			err,
			"NamespaceGathering",
			"error fetching list of namespaces from cluster")
	}
	_log.V(3).Info("calculated required namespaces", "requiredNamespaces", *requiredNamespaces)

	// parse through all namespaces and check each for the defined object
	for _, namespace := range namespaces.Items {
		// reconcile the object for a specific namespace, if an error occurs, then throw reconcile error
		if err := r.reconcileObjectForNamespace(
			log.IntoContext(ctx, _log.WithValues(
				"*clusterObject", *clusterObject,
				"namespace.GetName()", namespace.GetName(),
			)),
			clusterObject,
			namespace,
			requiredNamespaces); err != nil {

			return ctrl.Result{}, err
		}
	}

	_log.Info("reconciled")

	r.Recorder.Eventf(
		clusterObject,
		"Normal",
		"ReconciledObject",
		"successfully cloned resource in required namespaces")

	return ctrl.Result{}, r.setCondition(
		ctx,
		clusterObject,
		Condition_Ready,
		metav1.ConditionTrue,
		"DeployedResource",
		"successfully deployed resource [%s/%s:%s]",
		clusterObject.Replicator.Resource.GetAPIVersion(),
		clusterObject.Replicator.Resource.GetKind(),
		clusterObject.Replicator.Resource.GetName(),
	)
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
func (r *ClusterObjectReconciler) reconcileObjectForNamespace(
	ctx context.Context,
	clusterObject *clusterv1alpha1.ClusterObject,
	namespace corev1.Namespace,
	requiredNamespaces *corev1.NamespaceList) error {

	var _log = log.FromContext(ctx)

	_log.V(3).Info("check object")

	// create copy of resources object
	var typedObject = clusterObject.Replicator.Resource.DeepCopy()

	_log.V(5).Info("object from resources cached",
		"*typedObject", *typedObject,
		"clusterObject.Replicator.Resource", clusterObject.Replicator.Resource)

	// check, if the object does exist in the namespace and copy its content to cache
	doesExist, err := r.objectExists(ctx, namespace.GetName(), typedObject)
	if err != nil {
		return r.throwOnError(
			ctx,
			clusterObject,
			err,
			"ClusterObjectFetching",
			"error receiving the object from the cluster")
	}

	// check, if the object should exist in the namespace
	shouldExist := r.objectShouldExist(namespace, requiredNamespaces)
	_log.V(3).Info("state calculated", "shouldExist", shouldExist, "doesExist", doesExist)

	// after calculating the current state, handle the 4 cases
	if !shouldExist && !doesExist { // --------------------------------------------------------- case 1 -> ignore
		_log.V(3).Info("ignoring")
		return nil
	}

	if shouldExist && !doesExist { // --------------------------------------------------------- case 2 -> create
		_log.V(3).Info("creating")

		// create the new object, as a blueprint, to create it in the cluster
		typedObject := clusterObject.Replicator.Resource.DeepCopy()

		// change the namespace, to the requested namespace
		typedObject.SetNamespace(namespace.Name)

		// set the owners reference
		// this is required for watching the dependent objects
		if err := controllerutil.SetControllerReference(clusterObject, typedObject, r.Scheme); err != nil {
			return r.throwOnError(ctx, clusterObject, err, "OwnerReferenceConfiguration", "unable to set owners reference")
		}

		// create the object in the cluster
		if err := r.Create(ctx, typedObject, &client.CreateOptions{}); err != nil {
			return r.throwOnError(ctx, clusterObject, err, "ObjectCreation", "error creating object in namespace")
		}
	}

	// if the object does exist, and either should be updated or deleted,
	// check if the owner is in fact the clusterobject
	if !metav1.IsControlledBy(typedObject, clusterObject) {
		_log.V(3).Info("object does not contain ownerreference")
		return nil
	}

	if shouldExist && doesExist { // --------------------------------------------------------- case 3 -> update
		_log.V(3).Info("updating")
		// update the values of the tempObject
		typedObject = clusterObject.Replicator.Resource.DeepCopy()
		typedObject.SetNamespace(namespace.Name)

		// set the owners reference again
		// this is required for watching the dependent objects
		if err := controllerutil.SetControllerReference(clusterObject, typedObject, r.Scheme); err != nil {
			return r.throwOnError(ctx, clusterObject, err, "OwnerReferenceConfiguration", "unable to set owners reference")
		}

		// update the object
		if err := r.Update(ctx, typedObject, &client.UpdateOptions{}); err != nil {
			return r.throwOnError(ctx, clusterObject, err, "ObjectUpdate", "error updating object")
		}
	}

	if !shouldExist && doesExist { // --------------------------------------------------------- case 4 -> delete
		_log.V(3).Info("deleting")
		// delete the object
		if err := r.Delete(ctx, typedObject, &client.DeleteOptions{}); client.IgnoreNotFound(err) != nil {
			return r.throwOnError(ctx, clusterObject, err, "ObjectDeletion", "error deleting object")
		}
	}

	return nil
}
