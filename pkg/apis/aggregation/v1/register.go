package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const GroupName = "aggregation.open-cluster-management.io"

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}

var (
	SchemeBuilder    = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme      = SchemeBuilder.AddToScheme
	GroupVersionKind = schema.GroupVersionKind{
		Group:   GroupName,
		Version: "v1",
		Kind:    "ClusterStatus",
	}
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ClusterStatus{},
		&ClusterStatusList{},
		&ClusterStatusProxyOptions{},
	)
	return nil
}
