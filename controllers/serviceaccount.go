package controllers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *OnionServiceReconciler) torServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: *r.NewObjectMeta(),
	}
}

func (r *OnionServiceReconciler) ReconcileServiceAccount(req ctrl.Request) error {
	serviceAccount := r.torServiceAccount()
	found := &corev1.ServiceAccount{}

	err := r.Get(r.ctx, req.NamespacedName, found)
	if errors.IsNotFound(err) {
		err = r.Create(r.ctx, serviceAccount)
	}

	if err != nil {
		return err
	}

	//if !metav1.IsControlledBy(role, found) {
	//	msg := fmt.Sprintf(MessageResourceExists, role.Name)
	//	r.Recorder.Event(r.instance, corev1.EventTypeWarning, ErrResourceExists, msg)
	//	return fmt.Errorf(msg)
	//}

	if !reflect.DeepEqual(serviceAccount, found) {
		found.ObjectMeta = serviceAccount.ObjectMeta
		r.Log.Info("Updating Service Account %s/%s\n", serviceAccount.Namespace, serviceAccount.Name)
		err = r.Update(r.ctx, found)
	}

	return err
}