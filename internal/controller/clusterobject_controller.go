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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	"slices"

	clusterv1alpha1 "github.com/jnnkrdb/echosec/api/v1alpha1"
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
					for _, co := range list.Items {
						requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: co.Name}})
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

	ctx = co.IntoContext(ctx)

	labelselector, err := metav1.LabelSelectorAsSelector(co.LabelSelector)
	if err != nil {
		return ctrl.Result{}, r.throwOnError(ctx, err, "LabelSelectorFetching",
			"error fetching labelselector from clusterobject")
	}

	// request a list of namespaces, to parse through the list and
	// then check every namespace with the give item
	var namespaces = &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces, &client.ListOptions{LabelSelector: labelselector}); err != nil {
		return ctrl.Result{}, r.throwOnError(ctx, err, "NamespaceGathering",
			"error fetching list of namespaces from cluster")
	}

	for _, namespace := range namespaces.Items {
		// reconcile the object for a specific namespace, if an error occurs, then throw reconcile error
		if err := r.reconcileObjectForNamespace(ctx, co, namespace); err != nil {
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
	ctx context.Context, co *clusterv1alpha1.ClusterObject, namespace corev1.Namespace) error {

	var _log = log.FromContext(ctx).WithValues(
		"apiVersion", co.Resource.GetAPIVersion(),
		"kind", co.Resource.GetKind(),
		"name", co.Resource.GetName(),
		"namespace", namespace.GetName(),
	)

	_log.V(3).Info("check object")

	// check, if the object should exist in the namespace
	shouldExist, err := co.RegexRules.ShouldExistInNamespace(namespace.Name)
	if err != nil {
		return r.throwOnError(ctx, err, "NamespaceCalculating", "error calculating wether the item should exist or not")
	}

	var doesExist = false
	var typedObject = co.Resource.DeepCopy()
	// check, if the requested object does exist in the namespace
	if err := r.Get(ctx,
		types.NamespacedName{
			Namespace: namespace.Name,
			Name:      co.Resource.GetName(),
		}, typedObject, &client.GetOptions{}); err != nil {

		if client.IgnoreNotFound(err) != nil {
			return r.throwOnError(ctx, err, "ClusterObjectFetching", "error receiving the object from the cluster")
		}

	} else {

		doesExist = true
	}

	_log.V(3).Info("state calculated", "shouldExist", shouldExist, "doesExist", doesExist)

	// after calculating the current state, handle the 4 cases
	switch {
	case !shouldExist && !doesExist: // --------------------------------------------------------- case 1 -> ignore
		_log.V(3).Info("ignoring")

	case shouldExist && !doesExist: // --------------------------------------------------------- case 2 -> create
		return r.createObject(log.IntoContext(ctx, _log), co, namespace)

	case shouldExist && doesExist: // --------------------------------------------------------- case 3 -> update
		return r.updateObject(log.IntoContext(ctx, _log), co, typedObject, namespace)

	case !shouldExist && doesExist: // --------------------------------------------------------- case 4 -> delete
		return r.deleteObject(log.IntoContext(ctx, _log), co, typedObject)
	}

	return nil
}

// create the typedobject
func (r *ClusterObjectReconciler) createObject(
	ctx context.Context, co *clusterv1alpha1.ClusterObject, namespace corev1.Namespace) error {

	var _log = log.FromContext(ctx).WithValues()

	_log.V(3).Info("creating")

	// create the new object, as a blueprint, to create it in the cluster
	typedObject := co.Resource.DeepCopy()

	// change the namespace, to the requested namespace
	typedObject.SetNamespace(namespace.Name)

	// set the owners reference
	// this is required for watching the dependent objects
	if err := controllerutil.SetControllerReference(co, typedObject, r.Scheme); err != nil {
		return r.throwOnError(ctx, err, "OwnerReferenceConfiguration", "unable to set owners reference")
	}

	// create the object in the cluster
	if err := r.Create(ctx, typedObject, &client.CreateOptions{}); err != nil {
		return r.throwOnError(ctx, err, "ObjectCreation", "error creating object in namespace")
	}

	return nil
}

// update the typedobject, if it belongs to the given clusterobject
func (r *ClusterObjectReconciler) updateObject(
	ctx context.Context, co *clusterv1alpha1.ClusterObject, typedObject *unstructured.Unstructured, namespace corev1.Namespace) error {

	var _log = log.FromContext(ctx).WithValues(
		"apiVersion", typedObject.GetAPIVersion(),
		"kind", typedObject.GetKind(),
		"name", typedObject.GetName(),
		"namespace", typedObject.GetName(),
	)

	_log.V(3).Info("updating")

	// check if the found item belongs to the clusterobject reference
	if !slices.Contains(typedObject.GetOwnerReferences(), *metav1.NewControllerRef(co, co.GroupVersionKind())) {

		_log.V(3).Info("object does not contain ownerreference, blocking update",
			"ownerref.UID", co.GetUID(),
			"ownerref.Name", co.GetName(),
			"gvk", co.GroupVersionKind().String())

		return nil
	}

	// update the values of the tempObject
	typedObject = co.Resource.DeepCopy()
	typedObject.SetNamespace(namespace.Name)

	// set the owners reference again
	// this is required for watching the dependent objects
	if err := controllerutil.SetControllerReference(co, typedObject, r.Scheme); err != nil {
		return r.throwOnError(ctx, err, "OwnerReferenceConfiguration", "unable to set owners reference")
	}

	// update the object
	if err := r.Update(ctx, typedObject, &client.UpdateOptions{}); err != nil {
		return r.throwOnError(ctx, err, "ObjectUpdate", "error updating object")
	}

	return nil
}

// remove the typedobject, if it belongs to the given clusterobject
func (r *ClusterObjectReconciler) deleteObject(
	ctx context.Context, co *clusterv1alpha1.ClusterObject, typedObject *unstructured.Unstructured) error {

	var _log = log.FromContext(ctx).WithValues(
		"apiVersion", typedObject.GetAPIVersion(),
		"kind", typedObject.GetKind(),
		"name", typedObject.GetName(),
		"namespace", typedObject.GetName(),
	)

	_log.V(3).Info("deleting")

	// check if the found item belongs to the clusterobject reference
	if !slices.Contains(typedObject.GetOwnerReferences(), *metav1.NewControllerRef(co, co.GroupVersionKind())) {

		_log.V(3).Info("object does not contain ownerreference, blocking deletion",
			"ownerref.UID", co.GetUID(),
			"ownerref.Name", co.GetName(),
			"gvk", co.GroupVersionKind().String())

		return nil
	}

	// delete the object
	if err := r.Delete(ctx, typedObject, &client.DeleteOptions{}); client.IgnoreNotFound(err) != nil {
		return r.throwOnError(ctx, err, "ObjectDeletion", "error deleting object")
	}

	return nil
}
