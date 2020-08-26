package local

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/marcus-sa/tor-operator/pkg/config"
	torv1alpha1 "github.com/marcus-sa/tor-operator/api/v1alpha1"
)

type Controller struct {
	indexer      cache.Indexer
	queue        workqueue.RateLimitingInterface
	informer     cache.Controller
	localManager *Manager
	context		 context.Context
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller, localManager *Manager, context context.Context) *Controller {
	return &Controller{
		informer:     informer,
		indexer:      indexer,
		queue:        queue,
		localManager: localManager,
		context: 	  context,
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.sync(key.(string))
	c.handleErr(err, key)
	return true
}

func (c *Controller) sync(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		fmt.Printf("Fetching object with key %s from store failed with %v\n", key, err)
		return err
	}

	if !exists {
		fmt.Printf("OnionService %s does not exist anymore\n", key)
	} else {
		instance := obj.(*torv1alpha1.OnionService)

		torConfig, err := config.TorConfigForService(instance)
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
			fmt.Printf("updating onion config for %s/%s\n", instance.Namespace, instance.Name)

			err = ioutil.WriteFile("/run/tor/torfile", []byte(torConfig), 0644)
			if err != nil {
				fmt.Printf("Writing config failed with %v\n", err)
				return err
			}

			c.localManager.daemon.Reload()
		}

		err = c.updateOnionServiceStatus(instance)
		if err != nil {
			fmt.Printf("Updating status failed with %v\n", err)
			return err
		}
	}
	return nil
}

func (c *Controller) updateOnionServiceStatus(instance *torv1alpha1.OnionService) error {
	hostname, err := ioutil.ReadFile("/run/tor/service/hostname")
	if err != nil {
		fmt.Printf("Got this error when trying to find hostname: %v", err)
		hostname = []byte("")
	}

	newHostname := strings.TrimSpace(string(hostname))

	if newHostname != instance.Status.Hostname {
		instanceCopy := instance.DeepCopy()
		instanceCopy.Status.Hostname = newHostname

		_, err = c.localManager.clientset..OnionServices(instance.Namespace).Update(instanceCopy)
		return err
	}
	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		fmt.Printf("Error syncing onionservice %v: %v\n", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	fmt.Printf("Dropping onionservice %q out of the queue: %v\n", key, err)
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	fmt.Println("Starting controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	fmt.Println("Stopping controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
