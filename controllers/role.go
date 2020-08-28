package controllers

import (
	torv1alpha1 "github.com/marcus-sa/tor-operator/api/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *OnionServiceReconciler) torRole() *rbacv1.Role  {
	return &rbacv1.Role{
		ObjectMeta: *r.NewObjectMeta(),
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{torv1alpha1.GroupVersion.Group},
				Verbs: []string{"get", "list", "watch", "update", "patch"},
				Resources: []string{"onionservices"},
			},
			{
				APIGroups: []string{""},
				Verbs: []string{"create", "update", "patch"},
				Resources: []string{"events"},
			},
		},
	}
}

func (r *OnionServiceReconciler) ReconcileRole(req ctrl.Request) error  {
	role := r.torRole()
	found := &rbacv1.Role{}

	err := r.Get(r.ctx, req.NamespacedName, found)
	if errors.IsNotFound(err) {
		err = r.Create(r.ctx, role)
	}

	if err != nil {
		return err
	}

	//if !metav1.IsControlledBy(role, found) {
	//	msg := fmt.Sprintf(MessageResourceExists, role.Name)
	//	r.Recorder.Event(r.instance, corev1.EventTypeWarning, ErrResourceExists, msg)
	//	return fmt.Errorf(msg)
	//}

	if !reflect.DeepEqual(role, found) {
		found.ObjectMeta = role.ObjectMeta
		found.Rules = role.Rules
		r.Log.Info("Updating Role %s/%s\n", role.Namespace, role.Name)
		err = r.Update(r.ctx, found)
	}

	return err
}