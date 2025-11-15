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

package v1alpha1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// get the name, which should be used for the dependent items
func (sj ClusterSecret) GetDependentsName() string {
	if sj.Spec.SecretName == nil {
		return sj.Name
	}
	return *sj.Spec.SecretName
}

// handle deletion process
func (sj *ClusterSecret) Finalize(ctx context.Context, c client.Client) (bool, error) {
	var _log = log.FromContext(ctx)

	_log.Info("checking object deletion request")
	if sj.DeletionTimestamp.IsZero() {

		// check if the finalizer is set
		_log.Info("object is not requested to be deleted, adding finalizer if required")
		if !controllerutil.ContainsFinalizer(sj, Finalizer) {
			controllerutil.AddFinalizer(sj, Finalizer)
			if err := c.Update(ctx, sj); err != nil {
				_log.Error(err, "error updating object in cluster")
				return false, err
			}
		}

		return false, nil
	}

	// maybe this is not needed due to the owner references
	/*
		// create the delete options with the correct label selector
		var deleteAllOptions = &client.DeleteAllOfOptions{}
		deleteAllOptions.LabelSelector = ObjectsLabelSelector(sj.GetUID())

		// remove all items from the cluster
		if err := c.DeleteAllOf(ctx, &corev1.Secret{}, deleteAllOptions); err != nil {
			_log.Error(err, "error removing all objects from cluster with specific labelselector", "labelselector", deleteAllOptions.LabelSelector)
			return false, err
		}
	*/
	return false, nil
}
