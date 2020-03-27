package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"

	aggregationv1 "github.com/skeeey/aggregator-proxy-server/pkg/apis/aggregation/v1"
	"github.com/skeeey/aggregator-proxy-server/pkg/getter"
	"github.com/skeeey/aggregator-proxy-server/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
	proxyutil "k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/registry/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
)

// ProxyREST implements the proxy subresource for a Service
type AggregatorProxyRest struct {
	*getter.AggregatorServiceInfoGetter
}

func NewAggregatorProxyRest(serviceInfoGetter *getter.AggregatorServiceInfoGetter) *AggregatorProxyRest {
	return &AggregatorProxyRest{serviceInfoGetter}
}

var _ = rest.Connecter(&AggregatorProxyRest{})

// implement storage interface
func (r *AggregatorProxyRest) New() runtime.Object {
	return &aggregationv1.ClusterStatusProxyOptions{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *AggregatorProxyRest) ConnectMethods() []string {
	//TODO: may need more methods
	return []string{"GET", "POST", "PUT", "OPTIONS"}
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *AggregatorProxyRest) NewConnectOptions() (runtime.Object, bool, string) {
	return &aggregationv1.ClusterStatusProxyOptions{}, true, "path"
}

// Connect returns a handler for the pod proxy
func (r *AggregatorProxyRest) Connect(
	_ context.Context, _ string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	return &proxyRestHandler{opts: opts, responder: responder, serviceInfoGetter: r.AggregatorServiceInfoGetter}, nil
}

type proxyRestHandler struct {
	opts              runtime.Object
	responder         rest.Responder
	serviceInfoGetter *getter.AggregatorServiceInfoGetter
}

func (h *proxyRestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// the request URL must comply with the rules
	subResource, err := utils.GetSubResource(req.URL.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("the request %s is forbidden", req.URL.Path), http.StatusForbidden)
		return
	}

	serviceInfo := h.serviceInfoGetter.GetAggregatorServiceInfo(subResource)
	if serviceInfo == nil {
		klog.Warningf("The aggregator service cannot be found for %s", req.URL.Path)
		http.Error(w, fmt.Sprintf("the aggregator service (%s) is not found", subResource), http.StatusNotFound)
		return
	}

	transport, err := restclient.TransportFor(serviceInfo.RestConfig)
	if err != nil {
		klog.Errorf("failed to build transport for %s", serviceInfo.Name)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	host := fmt.Sprintf("%s.%s.svc", serviceInfo.ServiceName, serviceInfo.ServiceNamespace)
	proxyPath := serviceInfo.RootPath
	if serviceInfo.UseID {
		//TODO: find cluster name from req.URL.Path
		proxyPath = path.Join(proxyPath, "")
	}
	proxyOpts, ok := h.opts.(*aggregationv1.ClusterStatusProxyOptions)
	if !ok {
		klog.Errorf("invalid options object: %#v", h.opts)
		http.Error(w, "failed to get proxy path", http.StatusInternalServerError)
		return
	}
	proxyPath = path.Join("/", proxyPath, proxyOpts.Path)

	location := &url.URL{
		Scheme: "https", // should always be https
		Host:   net.JoinHostPort(host, serviceInfo.ServicePort),
		Path:   proxyPath,
	}
	klog.Infof("Proxy %s to %s", req.URL.Path, location.Path)
	proxyHandler := proxyutil.NewUpgradeAwareHandler(location, transport, true, false, proxyutil.NewErrorResponder(h.responder))
	proxyHandler.ServeHTTP(w, req)
}
