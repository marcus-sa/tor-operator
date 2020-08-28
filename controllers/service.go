package controllers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *OnionServiceReconciler) UpdateServiceStatus(req ctrl.Request) error {
	instanceCopy := r.instance.DeepCopy()
	service := &corev1.Service{}
	err := r.Get(r.ctx, req.NamespacedName, service)
	clusterIP := "None"
	if errors.IsNotFound(err) {
		clusterIP = "0.0.0.0"
	} else if err != nil {
		return err
	} else {
		clusterIP = service.Spec.ClusterIP
	}

	instanceCopy.Status.TargetClusterIP = clusterIP
	err = r.Update(r.ctx, instanceCopy)
	return err
}

func (r *OnionServiceReconciler) torService() (*corev1.Service, error) {
	var ports []corev1.ServicePort
	for _, p := range r.instance.Spec.Ports {
		port := corev1.ServicePort{
			Name:       p.Name,
			TargetPort: intstr.FromInt(int(p.TargetPort)),
			Port:       p.TargetPort,
		}
		ports = append(ports, port)
	}

	service := &corev1.Service{
		ObjectMeta: *r.NewObjectMeta(),
		Spec: corev1.ServiceSpec{
			Selector: r.instance.Spec.Selector,
			Ports:    ports,
		},
	}

	err := controllerutil.SetControllerReference(r.instance, service, r.Scheme)
	return service, err
}

func (r *OnionServiceReconciler) ReconcileService(req ctrl.Request) error {
	service, _ := r.torService()
	found := &corev1.Service{}

	if err := r.Get(r.ctx, req.NamespacedName, found); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(r.ctx, service); err != nil {
				return err
			}
		}

		return err
	}

	//if !metav1.IsControlledBy(service, found) {
	//	msg := fmt.Sprintf(MessageResourceExists, service.Name)
	//	r.Recorder.Event(r.instance, corev1.EventTypeWarning, ErrResourceExists, msg)
	//	return fmt.Errorf(msg)
	//}

	if !reflect.DeepEqual(service.Spec, found.Spec) {
		found.Spec = service.Spec
		r.Log.Info("Updating Service %s/%s\n", service.Namespace, service.Name)
		if err := r.Update(r.ctx, found); err != nil {
			return err
		}
	}

	return nil
}
