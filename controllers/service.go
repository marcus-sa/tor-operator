package controllers

import (
	"context"
	"fmt"
	torv1alpha1 "github.com/marcus-sa/tor-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *OnionServiceReconciler) UpdateServiceStatus(instance *torv1alpha1.OnionService, ctx context.Context, req ctrl.Request) error  {
	instanceCopy := instance.DeepCopy()
	service := &corev1.Service{}
	err := r.Get(ctx, req.NamespacedName, service)
	clusterIP := ""
	if errors.IsNotFound(err) {
		clusterIP = "0.0.0.0"
	} else if err != nil {
		return err
	} else {
		clusterIP = service.Spec.ClusterIP
	}

	instanceCopy.Status.TargetClusterIP = clusterIP
	err = r.Update(ctx, instanceCopy)
	return err
}

func (r *OnionServiceReconciler) torService(instance *torv1alpha1.OnionService) (*corev1.Service, error) {
	var ports []corev1.ServicePort
	for _, p := range instance.Spec.Ports {
		port := corev1.ServicePort{
			Name:       p.Name,
			TargetPort: intstr.FromInt(int(p.TargetPort)),
			Port:       p.TargetPort,
		}
		ports = append(ports, port)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, schema.GroupVersionKind{
					Group:   torv1alpha1.GroupVersion.Group,
					Version: torv1alpha1.GroupVersion.Version,
					Kind:    "OnionService",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: instance.Spec.Selector,
			Ports:    ports,
		},
	}

	err := controllerutil.SetControllerReference(instance, service, r.Scheme)
	return service, err
}

func (r *OnionServiceReconciler) ReconcileService(instance *torv1alpha1.OnionService, ctx context.Context, req ctrl.Request) error {
	service, _ := r.torService(instance)
	found := &corev1.Service{}

	if err := r.Get(ctx, req.NamespacedName, found); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, service); err != nil {
				return err
			}
		}

		return err
	}

	if !metav1.IsControlledBy(service, found) {
		msg := fmt.Sprintf(MessageResourceExists, service.Name)
		r.recorder.Event(instance, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	if !reflect.DeepEqual(service.Spec, found.Spec) {
		found.Spec = service.Spec
		//log.Printf("Updating Deployment %s/%s\n", service.Namespace, service.Name)
		if err := r.Update(ctx, found); err != nil {
			return err
		}
	}

	return nil
}