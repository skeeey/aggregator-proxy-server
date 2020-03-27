package options

import (
	"fmt"

	"github.com/skeeey/aggregator-proxy-server/pkg/api"
	"github.com/skeeey/aggregator-proxy-server/pkg/apis/aggregation/openapi"
	"github.com/spf13/pflag"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericapiserveroptions "k8s.io/apiserver/pkg/server/options"
)

type Options struct {
	KubeConfigFile string

	ServerRun      *genericapiserveroptions.ServerRunOptions
	SecureServing  *genericapiserveroptions.SecureServingOptionsWithLoopback
	Authentication *genericapiserveroptions.DelegatingAuthenticationOptions
	Authorization  *genericapiserveroptions.DelegatingAuthorizationOptions
}

// NewOptions constructs a new set of default options for aggregator-proxy-server.
func NewOptions() *Options {
	return &Options{
		ServerRun:      genericapiserveroptions.NewServerRunOptions(),
		SecureServing:  genericapiserveroptions.NewSecureServingOptions().WithLoopback(),
		Authentication: genericapiserveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericapiserveroptions.NewDelegatingAuthorizationOptions(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeConfigFile, "kube-config-file", "", "Kubernetes configuration file to connect to kube-apiserver")

	o.ServerRun.AddUniversalFlags(fs)
	o.SecureServing.AddFlags(fs)
	o.Authentication.AddFlags(fs)
	o.Authorization.AddFlags(fs)
}

func (o Options) APIServerConfig() (*genericapiserver.Config, error) {
	if err := o.ServerRun.DefaultAdvertiseAddress(o.SecureServing.SecureServingOptions); err != nil {
		return nil, err
	}
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts(o.ServerRun.AdvertiseAddress.String(), nil, nil); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewConfig(api.Codecs)
	if err := o.ServerRun.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, nil); err != nil {
		return nil, err
	}

	//TODO: add custormer authorization here
	if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
		return nil, err
	}

	// enable OpenAPI schemas
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(
		openapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(api.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "aggregator-proxy-server"
	serverConfig.OpenAPIConfig.Info.Version = "0.0.1"

	return serverConfig, nil
}
