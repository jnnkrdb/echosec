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

	clusterv1alpha1 "github.com/jnnkrdb/r8r/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// This function is used to handle the errors, whichget thrown by the reconciliation.
// It packs together the log, the conditions and the events.
//
// parameters:
//   - ctx context.Contex -> this is the default given context
//   - err error          -> this is the thrown error, which should be handled
func (r *ClusterObjectReconciler) throwOnError(ctx context.Context, err error, event, msg string) error {

	// if the error is in fact nil, then leave early
	if err == nil {
		return err
	}

	var _log = log.FromContext(ctx)

	// log the message with the error in the binary logs
	_log.Error(err, msg)

	var co = &clusterv1alpha1.ClusterObject{}
	if err := co.FromContext(ctx); err != nil {
		_log.Error(err, "error reading clusterobject from context")
		return err
	}

	// throw the event to the object
	r.Recorder.Eventf(co,
		"Warning",
		fmt.Sprintf("%sError", event),
		"%s: %v", msg, err,
	)

	// set the condition if any
	if e := co.SetCondition(ctx,
		r.Client,
		clusterv1alpha1.Condition_Ready,
		metav1.ConditionFalse,
		fmt.Sprintf("Failed%s", event),
		"%s: %v", msg, err,
	); e != nil {
		return e
	}
	return err
}

// validate wether an object is existing in a given namespace or not
func (r *ClusterObjectReconciler) objectExists(ctx context.Context, namespace string, typedObject *unstructured.Unstructured) (bool, error) {

	var _log = log.FromContext(ctx)

	var co = &clusterv1alpha1.ClusterObject{}
	if err := co.FromContext(ctx); err != nil {
		_log.Error(err, "error reading clusterobject from context")
		return false, err
	}

	// create a short local copy of the requested resource, for the api request
	typedObject = co.Resource.DeepCopy()

	// check, if the requested object does exist in the namespace
	if err := r.Get(ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      co.Resource.GetName(),
		}, typedObject, &client.GetOptions{}); err != nil {

		return false, client.IgnoreNotFound(err)
	}

	return true, nil
}

// validate wether an object is existing in a given namespace or not
func (r *ClusterObjectReconciler) objectShouldExist(namespace corev1.Namespace, requiredNamespaces *corev1.NamespaceList) bool {

	for _, checkingNamespace := range requiredNamespaces.Items {

		if checkingNamespace.GetName() == namespace.GetName() {

			return true
		}
	}

	return false
}
