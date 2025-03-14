package controller

import (
	"context"
	"time"

	"github.com/jnnkrdb/echosec/pkg/reconcilation/objects/configmap"
	"github.com/jnnkrdb/echosec/pkg/reconcilation/regx"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var _log = log.FromContext(ctx)
	var defaultResult = ctrl.Result{RequeueAfter: time.Duration(viper.GetInt("syncperiodminutes")) * time.Minute}

	_log.Info("got item", "name", req.Name, "namespace", req.Namespace)

	var srcConfigMap = &corev1.ConfigMap{}

	if err := r.Client.Get(ctx, req.NamespacedName, srcConfigMap, &client.GetOptions{}); err != nil {
		_log.Error(err, "error receiving configmap from cluster")
		return defaultResult, err
	}

	// handle finalization tasks
	if finalized, err := configmap.Finalize(ctx, r.Client, srcConfigMap); err != nil {
		return defaultResult, err
	} else if finalized {
		return ctrl.Result{}, nil
	}

	// clone the content of the object into other namespaces
	var existing_namespaces = &corev1.NamespaceList{}
	if err := r.Client.List(ctx, existing_namespaces, &client.ListOptions{}); err != nil {
		return defaultResult, err
	}

	for _, current_namespace := range existing_namespaces.Items {

		var (
			_tmpCM                = &corev1.ConfigMap{}
			_namespacedName       = types.NamespacedName{Namespace: current_namespace.Name, Name: srcConfigMap.Name}
			_tmpLog               = _log.WithValues("namespace", current_namespace.Name, "requested.cm", _namespacedName.String())
			_shouldExist    bool  = false
			_err            error = nil
		)

		_tmpLog.Info("checking namespace")

		// check if the item should exist in this namespace
		if _shouldExist, _err = regx.ShouldExistInNamespace(srcConfigMap.Annotations, current_namespace.Name); _err != nil {
			_tmpLog.Error(_err, "error calculating namespace existence")
			return defaultResult, _err
		}

		_err = r.Client.Get(ctx, _namespacedName, _tmpCM, &client.GetOptions{})

		switch {

		case _err == nil && !_shouldExist: // if it exists and should not exist, remove it from the cluster
			_tmpLog.Info("configmap should not exist in namespace -> deleting")

			if e := r.Client.Delete(ctx, _tmpCM, &client.DeleteOptions{}); e != nil {
				_tmpLog.Error(e, "error removing item from cluster")
				return defaultResult, e
			}

		case _err == nil && _shouldExist: // if it exists and should exist, update the item, if neccessary
			_tmpLog.Info("configmap does already exist in namespace -> updating")

			// TODO: implement an update configmap method

		case errors.IsNotFound(_err) && _shouldExist: // if the item does not exist in the current namespace, but should exist, create it
			_tmpLog.Info("configmap does not exist in namespace -> creating")

			_tmpCM.ObjectMeta = v1.ObjectMeta{
				Name:      srcConfigMap.Name,
				Namespace: srcConfigMap.Namespace,
			}

			// TODO: implement an create configmap method
		}
	}

	return defaultResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
