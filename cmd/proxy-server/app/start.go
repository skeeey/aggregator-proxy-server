package app

import (
	"time"

	"github.com/skeeey/aggregator-proxy-server/cmd/proxy-server/app/options"
	"github.com/skeeey/aggregator-proxy-server/pkg/controller"
	"github.com/skeeey/aggregator-proxy-server/pkg/getter"
	"github.com/skeeey/aggregator-proxy-server/pkg/server"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func Run(opts *options.Options, stopCh <-chan struct{}) error {
	clusterCfg, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfigFile)
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	serviceInfoGetter := getter.NewAggregatorServiceInfoGetter()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
	ctrl := controller.NewAggregatorServiceInfoController(kubeClient, informerFactory, serviceInfoGetter, stopCh)
	go ctrl.Run()
	informerFactory.Start(stopCh)

	apiServerConfig, err := opts.APIServerConfig()
	if err != nil {
		return err
	}
	proxyServer, err := server.NewProxyServer(informerFactory, apiServerConfig, serviceInfoGetter)
	if err != nil {
		return err
	}
	return proxyServer.Run(stopCh)
}
