package api

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/skeeey/aggregator-proxy-server/pkg/apis/aggregation"
	aggregationv1 "github.com/skeeey/aggregator-proxy-server/pkg/apis/aggregation/v1"
	"github.com/skeeey/aggregator-proxy-server/pkg/getter"
	"github.com/skeeey/aggregator-proxy-server/pkg/proxy"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/metrics"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

const resource = "clusterstatuses"

var (
	// Scheme contains the types needed by the resource metrics API.
	Scheme = runtime.NewScheme()
	// ParameterCodec handles versioning of objects that are converted to query parameters.
	ParameterCodec = runtime.NewParameterCodec(Scheme)
	// Codecs is a codec factory for serving the resource metrics API.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	aggregation.Install(Scheme)

	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

func Install(serviceInfoGetter *getter.AggregatorServiceInfoGetter, server *genericapiserver.GenericAPIServer) error {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(aggregationv1.GroupName, Scheme, ParameterCodec, Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[aggregationv1.SchemeGroupVersion.Version] = map[string]rest.Storage{
		resource: &clusterStatusStorage{},
		//"clusterstatuses/aggregator": proxy.NewAggregatorProxyRest(serviceInfoGetter),
	}

	return server.InstallAPIGroup(&apiGroupInfo)
}

type action struct {
	Verb   string
	Path   string
	Params []*restful.Parameter
}

const (
	ROUTE_META_GVK    = "x-kubernetes-group-version-kind"
	ROUTE_META_ACTION = "x-kubernetes-action"
)

func AddSubResource(subresource string, serviceInfoGetter *getter.AggregatorServiceInfoGetter, server *genericapiserver.GenericAPIServer) error {
	prefix := path.Join(discovery.APIGroupPrefix, aggregationv1.GroupName, aggregationv1.GroupVersionKind.Version)
	var ws *restful.WebService
	for _, _ws := range server.Handler.GoRestfulContainer.RegisteredWebServices() {
		if _ws.RootPath() == prefix {
			ws = _ws
		}
	}
	if ws == nil {
		return fmt.Errorf("Cannot find the web service for %s", prefix)
	}

	suffix := "/" + subresource
	itemPath := resource + "/{name}" + suffix
	params := []*restful.Parameter{}
	nameParams := append(params, ws.PathParameter("name", "name of the "+aggregationv1.GroupVersionKind.Kind).DataType("string"))
	proxyParams := append(nameParams, ws.PathParameter("path", "path to the resource").DataType("string"))

	actions := []action{}
	actions = append(actions, action{"CONNECT", itemPath, nameParams})
	actions = append(actions, action{"CONNECT", itemPath + "/{path:*}", proxyParams})

	reqScope := handlers.RequestScope{
		ParameterCodec: ParameterCodec,
		Namer: handlers.ContextBasedNaming{
			SelfLinker:         runtime.SelfLinker(meta.NewAccessor()),
			ClusterScoped:      true,
			SelfLinkPathPrefix: path.Join(prefix, resource) + "/",
			SelfLinkPathSuffix: suffix,
		},
		Kind: aggregationv1.GroupVersionKind,
	}

	connecter := proxy.NewAggregatorProxyRest(serviceInfoGetter)
	connectOptions, _, _ := connecter.NewConnectOptions()
	connectOptionsInternalKinds, _, err := Scheme.ObjectKinds(connectOptions)
	if err != nil {
		return err
	}
	versionedConnectOptions, err := Scheme.New(aggregationv1.GroupVersionKind.GroupVersion().WithKind(connectOptionsInternalKinds[0].Kind))
	if err != nil {
		return err
	}

	for _, action := range actions {
		routes := []*restful.RouteBuilder{}
		for _, method := range connecter.ConnectMethods() {
			requestScope := "cluster"
			operationSuffix := ""
			if strings.HasSuffix(action.Path, "/{path:*}") {
				requestScope = "resource"
				operationSuffix = operationSuffix + "WithPath"
			}

			handler := metrics.InstrumentRouteFunc(
				action.Verb,
				aggregationv1.GroupName,
				aggregationv1.GroupVersionKind.Version,
				resource,
				subresource,
				requestScope,
				metrics.APIServerComponent,
				func(req *restful.Request, res *restful.Response) {
					handlers.ConnectResource(connecter, &reqScope, nil, resource+"/"+subresource, true)(res.ResponseWriter, req.Request)
				},
			)

			route := ws.Method(method).Path(action.Path).
				To(handler).
				Doc("connect " + method + " requests to " + subresource + " of " + aggregationv1.GroupVersionKind.Kind).
				Operation("connect" + strings.Title(strings.ToLower(method)) + aggregationv1.GroupVersionKind.Kind + strings.Title(subresource) + operationSuffix).
				Produces("*/*").
				Consumes("*/*").
				Writes("string")

			if err := endpoints.AddObjectParams(ws, route, versionedConnectOptions); err != nil {
				return err
			}

			for _, param := range action.Params {
				route.Param(param)
			}
			routes = append(routes, route)
		}

		for _, route := range routes {
			route.Metadata(ROUTE_META_GVK, metav1.GroupVersionKind{
				Group:   aggregationv1.GroupVersionKind.Group,
				Version: aggregationv1.GroupVersionKind.Version,
				Kind:    aggregationv1.GroupVersionKind.Kind,
			})
			route.Metadata(ROUTE_META_ACTION, strings.ToLower(action.Verb))
			ws.Route(route)
		}
	}
	return nil
}

type clusterStatusStorage struct{}

var (
	_ = rest.Storage(&clusterStatusStorage{})
	_ = rest.KindProvider(&clusterStatusStorage{})
	_ = rest.Lister(&clusterStatusStorage{})
	_ = rest.Getter(&clusterStatusStorage{})
	_ = rest.Scoper(&clusterStatusStorage{})
)

// Storage interface
func (s *clusterStatusStorage) New() runtime.Object {
	return &aggregationv1.ClusterStatus{}
}

// KindProvider interface
func (s *clusterStatusStorage) Kind() string {
	return "ClusterStatus"
}

// Lister interface
func (s *clusterStatusStorage) NewList() runtime.Object {
	return &aggregationv1.ClusterStatusList{}
}

// Lister interface
func (s *clusterStatusStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return &aggregationv1.ClusterStatusList{}, nil
}

// Getter interface
func (s *clusterStatusStorage) Get(ctx context.Context, name string, opts *metav1.GetOptions) (runtime.Object, error) {
	return &aggregationv1.ClusterStatus{}, nil
}

// Scoper interface
func (s *clusterStatusStorage) NamespaceScoped() bool {
	return false
}
