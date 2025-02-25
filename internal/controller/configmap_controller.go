package controller

import (
	"context"
	"time"

	"github.com/jnnkrdb/echosec/pkg/finalization"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	if err := finalization.Check(ctx, r.Client, srcConfigMap); err != nil {
		return defaultResult, err
	}

	// finalize if requested

	if finalized, err := finalization.Finalize(ctx, r.Client, srcConfigMap, func() ([]client.Object, error) {
		var configmaps = &corev1.ConfigMapList{}
		if err := r.Client.List(ctx, configmaps, &client.ListOptions{}); err != nil {
			return []client.Object{}, err
		}
		return []client.Object(configmaps.Items), nil
	}); err != nil || finalized {
		return defaultResult, err
	}

	if srcConfigMap.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(srcConfigMap, finalization.Finalizer) {

		for _, item := range configmaps.Items {

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
