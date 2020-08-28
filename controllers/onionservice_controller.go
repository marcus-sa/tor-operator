/*


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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	torv1alpha1 "github.com/marcus-sa/tor-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"
	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Foo"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Foo synced successfully"
)

// OnionServiceReconciler reconciles a OnionService object
type OnionServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder

	ctx      context.Context
	instance *torv1alpha1.OnionService
}

func (r *OnionServiceReconciler) NewObjectMeta() *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Name: r.instance.Name,
		Namespace: r.instance.Namespace,
		OwnerReferences: []metav1.OwnerReference{
			*r.NewOwnerReference(),
		},
	}
}

func (r *OnionServiceReconciler) NewOwnerReference() *metav1.OwnerReference {
	return metav1.NewControllerRef(r.instance, schema.GroupVersionKind{
		Group:   torv1alpha1.GroupVersion.Group,
		Version: torv1alpha1.GroupVersion.Version,
		Kind:    "OnionService",
	})
}

// +kubebuilder:informers:group=core,version=v1,kind=Pod
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:controller:group=tor,version=v1alpha1,kind=OnionService,resource=onionservices
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:informers:group=apps,version=v1,kind=Deployment
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:informers:group=core,version=v1,kind=Service
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:informers:group=core,version=v1,kind=Secret
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:informers:group=core,version=v1,kind=ServiceAccount
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
func (r *OnionServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.ctx = context.Background()
	log := r.Log.WithValues(req.Name, req.NamespacedName)

	r.instance = &torv1alpha1.OnionService{}

	if err := r.Get(r.ctx, req.NamespacedName, r.instance); err != nil {
		log.Error(err, "unable to fetch OnionService")

		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	var errs []error

	if err := r.ReconcileServiceAccount(req); err != nil {
		errs = append(errs, err)
		//return ctrl.Result{}, err
	}

	if err := r.ReconcileRole(req); err != nil {
		errs = append(errs, err)
		//return ctrl.Result{}, err
	}

	if err := r.ReconcileRoleBinding(req); err != nil {
		errs = append(errs, err)
		//return ctrl.Result{}, err
	}

	if err := r.ReconcileService(req); err != nil {
		errs = append(errs, err)
		//return ctrl.Result{}, err
	}

	if err := r.ReconcileDeployment(req); err != nil {
		errs = append(errs, err)
		//return ctrl.Result{}, err
	}

	// Finally, we update the status block of the OnionService resource to reflect the
	// current state of the world
	if err := r.UpdateServiceStatus(req); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(r.instance, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)

	return ctrl.Result{}, nil
}

func (r *OnionServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&torv1alpha1.OnionService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
