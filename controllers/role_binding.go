package controllers

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *OnionServiceReconciler) torRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: *r.NewObjectMeta(),
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: r.instance.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: r.instance.Name,
		},
	}
}

func (r *OnionServiceReconciler) ReconcileRoleBinding(req ctrl.Request) error  {
	roleBinding := r.torRoleBinding()
	found := &rbacv1.RoleBinding{}

	err := r.Get(r.ctx, req.NamespacedName, found)
	if errors.IsNotFound(err) {
		err = r.Create(r.ctx, roleBinding)
	}

	if err != nil {
		return err
	}

	//if !metav1.IsControlledBy(roleBinding, found) {
	//	msg := fmt.Sprintf(MessageResourceExists, roleBinding.Name)
	//	r.Recorder.Event(r.instance, corev1.EventTypeWarning, ErrResourceExists, msg)
	//	return fmt.Errorf(msg)
	//}

	if !reflect.DeepEqual(roleBinding, found) {
		found.ObjectMeta = roleBinding.ObjectMeta
		found.Subjects = roleBinding.Subjects
		found.RoleRef = roleBinding.RoleRef
		r.Log.Info("Updating RoleBinding %s/%s\n", roleBinding.Namespace, roleBinding.Name)
		err = r.Update(r.ctx, found)
	}

	return err
}