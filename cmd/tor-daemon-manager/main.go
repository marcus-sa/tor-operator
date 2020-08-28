package main

import (
	"flag"
	"fmt"
	torv1alpha1 "github.com/marcus-sa/tor-operator/api/v1alpha1"
	"github.com/marcus-sa/tor-operator/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	onionServiceNamespace string
	metricsAddr string
	onionServiceName string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(torv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	flag.StringVar(&onionServiceNamespace, "namespace", "",
		"The namespace of the OnionService to manage.")
	flag.StringVar(&onionServiceName, "name", "",
		"The name of the OnionService to manage.")
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
}

func main() {
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	var errs []error

	if onionServiceName == "" {
		errs = append(errs, fmt.Errorf("--name flag cannot be empty"))
	}
	if onionServiceNamespace == "" {
		errs = append(errs, fmt.Errorf("--namespace flag cannot be empty"))
	}
	if err := errors.NewAggregate(errs); err != nil {
		ctrl.Log.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.TorDaemonReconciler{
		Client:                mgr.GetClient(),
		Log:                   ctrl.Log.WithName("controllers").WithName("TorDaemon"),
		Scheme:                mgr.GetScheme(),
		OnionServiceName:      onionServiceName,
		OnionServiceNamespace: onionServiceNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TorDaemon")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
