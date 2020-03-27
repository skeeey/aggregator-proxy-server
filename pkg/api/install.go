package api

import (
	"context"

	"github.com/skeeey/aggregator-proxy-server/pkg/apis/aggregation"
	aggregationv1 "github.com/skeeey/aggregator-proxy-server/pkg/apis/aggregation/v1"
	"github.com/skeeey/aggregator-proxy-server/pkg/getter"
	"github.com/skeeey/aggregator-proxy-server/pkg/proxy"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

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
		"clusterstatuses":            &clusterStatusStorage{},
		"clusterstatuses/aggregator": proxy.NewAggregatorProxyRest(serviceInfoGetter),
	}

	return server.InstallAPIGroup(&apiGroupInfo)
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
