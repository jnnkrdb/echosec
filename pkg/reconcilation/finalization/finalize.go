package finalization

import (
	"context"

	"github.com/jnnkrdb/echosec/pkg/reconcilation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const Finalizer string = "echosec.jnnkrdb.de/finalizer"

// finalize the given object
func Finalize(ctx context.Context, c client.Client, obj client.Object) (bool, error) {
	var _log = log.FromContext(ctx).WithValues("kind", obj.GetObjectKind().GroupVersionKind().Kind)

	// checking the object for the finalizer, if the finalizer does not exist
	// then append it to the object
	if !controllerutil.ContainsFinalizer(obj, Finalizer) {
		_log.Info("appending the finalizer to the object")

		controllerutil.AddFinalizer(obj, Finalizer)

		if err := c.Update(ctx, obj, &client.UpdateOptions{}); err != nil {
			_log.Error(err, "error adding finalizer")
			return false, err
		}
	}

	// finalize items, if neccessary
	if obj.GetDeletionTimestamp() == nil {
		return false, nil
	}

	// create the delete options with the correct label selector
	var deleteAllOptions = &client.DeleteAllOfOptions{}
	deleteAllOptions.LabelSelector = reconcilation.ObjectsLabelSelector(obj.GetUID())

	// remove all items from the cluster
	if err := c.DeleteAllOf(ctx, obj, deleteAllOptions); err != nil {
		_log.Error(err, "error removing all objects from cluster with specific labelselector", "labelselector", deleteAllOptions.LabelSelector)
		return false, err
	}

	// after finalization remove the finalizer from the object
	controllerutil.RemoveFinalizer(obj, Finalizer)
	if err := c.Update(ctx, obj, &client.UpdateOptions{}); err != nil {
		_log.Error(err, "error removing finalizer")
		return false, err
	}

	return true, nil
}
