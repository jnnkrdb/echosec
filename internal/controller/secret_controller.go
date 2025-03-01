package controller

import (
	"context"
	"time"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Secret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var _log = log.FromContext(ctx)
	var defaultResult = ctrl.Result{RequeueAfter: time.Duration(viper.GetInt("syncperiodminutes")) * time.Minute}

	_log.Info("got item", "name", req.Name, "namespace", req.Namespace)

	var srcSecret = &corev1.Secret{}

	if err := r.Client.Get(ctx, req.NamespacedName, srcSecret, &client.GetOptions{}); err != nil {
		_log.Error(err, "error receiving secret from cluster")
		return defaultResult, err
	}

	_log.Info("received secret", "secret.uid", srcSecret.UID)

	return defaultResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
