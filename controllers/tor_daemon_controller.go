package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	torv1alpha1 "github.com/marcus-sa/tor-operator/api/v1alpha1"
	"github.com/marcus-sa/tor-operator/pkg/config"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"syscall"
	"time"
)

type TorDaemonReconciler struct {
	client.Client
	Log                   logr.Logger
	Scheme                *runtime.Scheme
	OnionServiceNamespace string
	OnionServiceName      string
	cmd                   *exec.Cmd
	ctx                   context.Context
	instance              *torv1alpha1.OnionService
	//metricsExporter 	  *metrics.TorDaemonMetricsExporter
}

func (r *TorDaemonReconciler) start() {
	go func() {
		for {
			fmt.Println("Starting tor...")
			r.cmd = exec.CommandContext(r.ctx,
				"tor",
				"-f", "/run/tor/torfile",
				"--allow-missing-torrc",
			)
			r.cmd.Stdout = os.Stdout
			r.cmd.Stderr = os.Stderr

			err := r.cmd.Start()
			if err != nil {
				fmt.Print(err)
			}
			r.cmd.Wait()
			time.Sleep(time.Second * 1)
		}
	}()
}

func (r *TorDaemonReconciler) reload() {
	fmt.Println("Reloading Tor daemon...")

	// start if not already running
	if r.cmd == nil || (r.cmd.ProcessState != nil && r.cmd.ProcessState.Exited()) {
		r.start()
	} else {
		r.cmd.Process.Signal(syscall.SIGHUP)
	}
}

func (r *TorDaemonReconciler) syncOnionConfig() error {
	torConfig, err := config.CreateTorConfigForService(r.instance)
	if err != nil {
		fmt.Printf("Generating config failed with %v\n", err)
		return err
	}

	reload := false

	torfile, err := ioutil.ReadFile("/run/tor/torfile")
	if os.IsNotExist(err) {
		reload = true
	} else if err != nil {
		return err
	}

	if string(torfile) != torConfig {
		reload = true
	}

	if reload {
		fmt.Printf("updating onion config for %s/%s\n", r.instance.Namespace, r.instance.Name)

		err = ioutil.WriteFile("/run/tor/torfile", []byte(torConfig), 0644)
		if err != nil {
			fmt.Printf("Writing config failed with %v\n", err)
			return err
		}

		r.reload()
	}

	err = r.updateOnionServiceStatus()
	if err != nil {
		fmt.Printf("Updating status failed with %v\n", err)
		return err
	}

	return nil
}

func (r *TorDaemonReconciler) updateOnionServiceStatus() error {
	hostname, err := ioutil.ReadFile("/run/tor/service/hostname")
	if err != nil {
		fmt.Printf("Got this error when trying to find hostname: %v", err)
		hostname = []byte("")
	}

	newHostname := strings.TrimSpace(string(hostname))

	if newHostname != r.instance.Status.Hostname {
		instanceCopy := r.instance.DeepCopy()
		instanceCopy.Status.Hostname = newHostname

		err = r.Update(r.ctx, instanceCopy)
		return err
	}
	return nil
}

func (r *TorDaemonReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	defer cancel()

	// Watch ReplicaSets and enqueue ReplicaSet object key
	//if err := r.Watch(&source.Kind{Type: &torv1alpha1.OnionService{}}, &handler.EnqueueRequestForObject{}); err != nil {
	//	r.Log.Error(err, "unable to watch OnionServices")
	//	os.Exit(1)
	//}

	r.instance = &torv1alpha1.OnionService{}

	err := r.Get(ctx, types.NamespacedName{Name: r.OnionServiceName, Namespace: r.OnionServiceNamespace}, r.instance)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(nil, "Could not find existing OnionService")
		} else if err := r.syncOnionConfig(); err != nil {
			r.Log.Error(err, "Unable to start Tor")
			os.Exit(1)
		}
	}

	// FIXME: find out how to filter the watch
	if r.instance.Name != req.Name {
		return ctrl.Result{}, nil
	}

	if r.instance.Namespace != req.Namespace {
		return ctrl.Result{}, nil
	}

	if err != nil {
		r.Log.Error(err, "Could not fetch OnionService")
		return ctrl.Result{}, err
	}

	if err := r.syncOnionConfig(); err != nil {
		return ctrl.Result{}, err
	}

	//metrics.TorDaemonMetricsExporter.Start()

	return ctrl.Result{}, r.syncOnionConfig()
}

func (r *TorDaemonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&torv1alpha1.OnionService{}).Complete(r)
}
