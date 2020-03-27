package controller

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/skeeey/aggregator-proxy-server/pkg/getter"
	"github.com/skeeey/aggregator-proxy-server/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type AggregatorServiceInfoController struct {
	serviceInfoGetter *getter.AggregatorServiceInfoGetter
	client            kubernetes.Interface
	informerFactory   informers.SharedInformerFactory
	lister            v1.ConfigMapLister
	synced            cache.InformerSynced
	workqueue         workqueue.RateLimitingInterface
	stopCh            <-chan struct{}
}

func NewAggregatorServiceInfoController(
	client kubernetes.Interface,
	informerFactory informers.SharedInformerFactory,
	serviceInfoGetter *getter.AggregatorServiceInfoGetter,
	stopCh <-chan struct{}) *AggregatorServiceInfoController {
	configMapInformer := informerFactory.Core().V1().ConfigMaps()

	controller := &AggregatorServiceInfoController{
		serviceInfoGetter: serviceInfoGetter,
		client:            client,
		informerFactory:   informerFactory,
		lister:            configMapInformer.Lister(),
		synced:            configMapInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "aggregatorServiceInfoController"),
		stopCh:            stopCh,
	}

	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueue(newObj)
		},
		DeleteFunc: controller.deleteObj,
	})

	return controller
}

var aggregatorConfigMapLabels = map[string]string{
	"config": "mcm-aggregator",
}

func (c *AggregatorServiceInfoController) enqueue(obj interface{}) {
	var key string
	var err error
	labelSelector := &metav1.LabelSelector{
		MatchLabels: aggregatorConfigMapLabels,
	}
	aggregatorConfigmap := obj.(*corev1.ConfigMap)
	if utils.MatchLabelForLabelSelector(aggregatorConfigmap.GetLabels(), labelSelector) {
		if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
			utilruntime.HandleError(err)
			return
		}
		c.workqueue.Add(key)
	}
}

func (c *AggregatorServiceInfoController) deleteObj(obj interface{}) {
	var ok bool
	if _, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		_, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
	}
	c.enqueue(obj)
}

func (c *AggregatorServiceInfoController) Run() {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Waiting for aggregator service configmap informer caches to sync")
	if !cache.WaitForCacheSync(c.stopCh, c.synced) {
		klog.Errorf("failed to wait for aggregator service configmap informer caches to sync")
		return
	}

	go wait.Until(c.runWorker, time.Second, c.stopCh)
	<-c.stopCh
	klog.Info("Shutting aggregator service info controller")
}

func (c *AggregatorServiceInfoController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *AggregatorServiceInfoController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced aggregator service configmap '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *AggregatorServiceInfoController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: '%s'", key))
		return nil
	}

	aggregatorConfigMap, err := c.lister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// configmap is deleted, delete aggregator config
			c.serviceInfoGetter.RemoveAggregatorServiceInfo(namespace + "/" + name)
			return nil
		}
		return err
	}

	aggregatorServiceInfo, err := c.generateAggregatorServiceInfo(aggregatorConfigMap)
	if err != nil {
		return err
	}

	c.serviceInfoGetter.AddAggregatorServiceInfo(aggregatorServiceInfo)
	return nil
}

var aggregatorOptionsKey = []string{"service", "port", "path", "sub-resource", "use-id", "secret"}

func (c *AggregatorServiceInfoController) generateAggregatorServiceInfo(cm *corev1.ConfigMap) (*getter.AggregatorServiceInfo, error) {
	for _, key := range aggregatorOptionsKey {
		if _, ok := cm.Data[key]; !ok {
			return nil, fmt.Errorf("the '%s' key is required in configmap %s/%s", key, cm.Namespace, cm.Name)
		}
	}

	serviceNamespace, serviceName, err := cache.SplitMetaNamespaceKey(cm.Data["service"])
	if err != nil {
		return nil, fmt.Errorf("the service format is wrong in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}

	secretNamespace, secretName, err := cache.SplitMetaNamespaceKey(cm.Data["secret"])
	if err != nil {
		return nil, fmt.Errorf("the secret format is wrong in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}

	if secretNamespace == "" {
		secretNamespace = serviceNamespace
	}

	secret, err := c.client.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}

	useID := false
	if cm.Data["use-id"] == "true" {
		useID = true
	}

	return &getter.AggregatorServiceInfo{
		Name:             cm.Namespace + "/" + cm.Name,
		SubResource:      strings.Trim(cm.Data["sub-resource"], "/"),
		ServiceName:      serviceName,
		ServiceNamespace: serviceNamespace,
		ServicePort:      cm.Data["port"],
		RootPath:         strings.Trim(cm.Data["path"], "/"),
		UseID:            useID,
		RestConfig: &rest.Config{
			TLSClientConfig: rest.TLSClientConfig{
				CertData: secret.Data["tls.crt"],
				KeyData:  secret.Data["tls.key"],
				CAData:   secret.Data["ca.crt"],
			},
		},
	}, nil
}
