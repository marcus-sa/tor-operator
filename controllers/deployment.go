package controllers

import (
	torv1alpha1 "github.com/marcus-sa/tor-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	privateKeyVolume = "private-key"
	torConfigVolume  = "tor-config"
	imageName        = "quay.io/tor-operator/daemon-manager:latest"
)

func (r *OnionServiceReconciler) torDeployment() (*appsv1.Deployment, error) {
	labels := map[string]string{
		"app": "tor",
		"api": "tor",
		"controller": r.instance.Name,
	}

	privateKeyMountPath := "/run/tor/service/hs_ed25519_secret_key"
	if r.instance.Spec.Version == 2 {
		privateKeyMountPath = "/run/tor/service/private_key"
	}

	// allow not specifying a private key
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount

	if r.instance.Spec.PrivateKeySecret != (torv1alpha1.SecretReference{}) {
		volumes = []corev1.Volume{
			{
				Name: privateKeyVolume,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: r.instance.Spec.PrivateKeySecret.Name,
					},
				},
			},
		}

		volumeMounts = []corev1.VolumeMount{
			{
				Name:      privateKeyVolume,
				MountPath: privateKeyMountPath,
				SubPath:   r.instance.Spec.PrivateKeySecret.Key,
			},
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: *r.NewObjectMeta(),
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tor",
							Image: imageName,
							Args: []string{
								"--name",
								r.instance.Name,
								"--namespace",
								r.instance.Namespace,
							},
							ImagePullPolicy: "IfNotPresent",

							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(r.instance, deployment, r.Scheme)
	return deployment, err
}

func (r *OnionServiceReconciler) ReconcileDeployment(req ctrl.Request) error {
	deployment, _ := r.torDeployment()
	found := &appsv1.Deployment{}

	if err := r.Get(r.ctx, req.NamespacedName, found); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(r.ctx, deployment); err != nil {
				return err
			}
		}

		return err
	}

	//if !metav1.IsControlledBy(deployment, found) {
	//	msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
	//	r.Recorder.Event(r.instance, corev1.EventTypeWarning, ErrResourceExists, msg)
	//	return fmt.Errorf(msg)
	//}

	if !reflect.DeepEqual(deployment.Spec, found.Spec) {
		found.Spec = deployment.Spec
		r.Log.Info("Updating Deployment %s/%s\n", deployment.Namespace, deployment.Name)
		if err := r.Update(r.ctx, found); err != nil {
			return err
		}
	}

	return nil
}
