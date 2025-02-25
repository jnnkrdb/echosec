package finalization

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const Finalizer string = "echosec.jnnkrdb.de/finalizer"

// check finalization
func Check(ctx context.Context, c client.Client, obj client.Object) error {
	var _log = log.FromContext(ctx).WithValues("kind", obj.GetObjectKind().GroupVersionKind().Kind)

	// checking the object for the finalizer, if the finalizer does not exist
	// then append it to the object
	if !controllerutil.ContainsFinalizer(obj, Finalizer) {
		_log.Info("appending the finalizer to the object")

		controllerutil.AddFinalizer(obj, Finalizer)

		if err := c.Update(ctx, obj, &client.UpdateOptions{}); err != nil {
			_log.Error(err, "error adding finalizer")
			return err
		}
	}

	return nil
}

// check the objects finalization status
func Finalize(ctx context.Context, c client.Client, obj client.Object, fnz func() ([]client.Object, error)) (bool, error) {
	var _log = log.FromContext(ctx).WithValues("kind", obj.GetObjectKind().GroupVersionKind().Kind)

	if obj.GetDeletionTimestamp() != nil &&
		controllerutil.ContainsFinalizer(obj, Finalizer) {

		_log.Info("finalizing related objects")

		// receive related objects
		if err := c.List(ctx, objList, &client.ListOptions{}); err != nil {
			return false, err
		}

		// parse through list and check every item for required annotation
		// and source information
		for _, item := range objList {

		}
	}

	return false, nil
}
