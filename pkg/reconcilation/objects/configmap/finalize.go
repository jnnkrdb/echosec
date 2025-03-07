package configmap

import (
	"context"

	"github.com/jnnkrdb/echosec/pkg/reconcilation"
	"github.com/jnnkrdb/echosec/pkg/reconcilation/finalization"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// finalize a given configmap, if required
func Finalize(ctx context.Context, c client.Client, _cm *corev1.ConfigMap) (bool, error) {
	var _log = log.FromContext(ctx)

	// check if the obj contains the finalizer
	if err := finalization.Check(ctx, c, _cm); err != nil {
		return false, err
	}

	// finalize configmaps
	if _cm.GetDeletionTimestamp() == nil {
		return false, nil
	}

	var (
		configmaps *corev1.ConfigMapList = &corev1.ConfigMapList{}
		successful bool                  = false
	)

	if err := c.List(ctx, configmaps, &client.ListOptions{
		LabelSelector: reconcilation.ObjectsLabelSelector(_cm.GetUID()),
	}); err != nil {
		return false, err
	}

	for _, item := range configmaps.Items {
		var _tmpLog = _log.WithValues("namespace", item.Namespace, "name", item.Name)
		_tmpLog.Info("checking configmap")

		var _tmpCM = &corev1.ConfigMap{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(&item), _tmpCM, &client.GetOptions{}); err != nil {
			_log.Error(err, "error receiving configmap from cluster")
			return false, err
		}

		_tmpLog.Info("removing object", "uid", _tmpCM.GetUID())

		// TODO: implement code for finalization

	}

	// after finalization remove the finalizer from the object
	if successful {
		controllerutil.RemoveFinalizer(_cm, finalization.Finalizer)
		if err := c.Update(ctx, _cm, &client.UpdateOptions{}); err != nil {
			_log.Error(err, "error removing finalizer")
			return false, err
		}
	}

	return true, nil
}
