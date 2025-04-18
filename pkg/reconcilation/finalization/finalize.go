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
func Finalize(ctx context.Context, c client.Client, obj client.Object, objType uint) (bool, error) {
	var _log = log.FromContext(ctx).WithValues("objType", objType, "kind", obj.GetObjectKind().GroupVersionKind().Kind)

	// ignore object, which are whether copy nor source
	if objType == reconcilation.ObjectIsNONE {
		// if the object already has a finalizer, then remove it
		if controllerutil.ContainsFinalizer(obj, Finalizer) {
			controllerutil.RemoveFinalizer(obj, Finalizer)
			if err := c.Update(ctx, obj, &client.UpdateOptions{}); err != nil {
				_log.Error(err, "error removing unneccessary finalizer")
				return false, err
			}
		}
		return false, nil
	}

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

	// if the object is a source object, all related objects
	// must be removed from the cluster, if not, then only the
	// finalizer will be removed from the copy object, so it can
	// be terminated.
	// If the copy object is still required in the cluster, and the#
	// corresponding namespace exists, then it will be recreated in the
	// next reconcilation run
	if objType == reconcilation.ObjectIsSOURCE {

		// create the delete options with the correct label selector
		var deleteAllOptions = &client.DeleteAllOfOptions{}
		deleteAllOptions.LabelSelector = reconcilation.ObjectsLabelSelector(obj.GetUID())

		// remove all items from the cluster
		if err := c.DeleteAllOf(ctx, obj, deleteAllOptions); err != nil {
			_log.Error(err, "error removing all objects from cluster with specific labelselector", "labelselector", deleteAllOptions.LabelSelector)
			return false, err
		}
	}

	// after finalization remove the finalizer from the object
	controllerutil.RemoveFinalizer(obj, Finalizer)
	if err := c.Update(ctx, obj, &client.UpdateOptions{}); err != nil {
		_log.Error(err, "error removing finalizer")
		return false, err
	}

	return true, nil
}
