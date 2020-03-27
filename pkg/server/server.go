package server

import (
	"github.com/skeeey/aggregator-proxy-server/pkg/api"
	"github.com/skeeey/aggregator-proxy-server/pkg/getter"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/informers"
)

type ProxyServer struct {
	*genericapiserver.GenericAPIServer
}

func NewProxyServer(
	informerFactory informers.SharedInformerFactory,
	apiServerConfig *genericapiserver.Config,
	serviceInfoGetter *getter.AggregatorServiceInfoGetter) (*ProxyServer, error) {
	apiServer, err := apiServerConfig.Complete(informerFactory).New("aggregator-proxy-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	if err := api.Install(serviceInfoGetter, apiServer); err != nil {
		return nil, err
	}

	return &ProxyServer{apiServer}, nil
}

func (p *ProxyServer) Run(stopCh <-chan struct{}) error {
	return p.GenericAPIServer.PrepareRun().Run(stopCh)
}
