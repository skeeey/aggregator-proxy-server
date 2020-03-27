package getter

import (
	"reflect"
	"sync"

	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

type AggregatorServiceInfo struct {
	Name             string
	SubResource      string
	ServiceName      string
	ServiceNamespace string
	ServicePort      string
	RootPath         string
	UseID            bool
	RestConfig       *rest.Config
}

type AggregatorServiceInfoGetter struct {
	mutex        sync.RWMutex
	serviceInfos map[string]*AggregatorServiceInfo
}

func NewAggregatorServiceInfoGetter() *AggregatorServiceInfoGetter {
	return &AggregatorServiceInfoGetter{
		serviceInfos: make(map[string]*AggregatorServiceInfo),
	}
}

func (g *AggregatorServiceInfoGetter) GetAggregatorServiceInfo(subResource string) *AggregatorServiceInfo {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return g.serviceInfos[subResource]
}

func (g *AggregatorServiceInfoGetter) AddAggregatorServiceInfo(serviceInfo *AggregatorServiceInfo) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if old, existed := g.serviceInfos[serviceInfo.SubResource]; existed {
		if !reflect.DeepEqual(old, serviceInfo) {
			klog.Infof("Update aggregator service info %s", serviceInfo.Name)
			g.serviceInfos[serviceInfo.SubResource] = serviceInfo
		}
		return
	}

	klog.Infof("Add aggregator service info %s", serviceInfo.Name)
	g.serviceInfos[serviceInfo.SubResource] = serviceInfo
}

func (g *AggregatorServiceInfoGetter) RemoveAggregatorServiceInfo(serviceInfoName string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for key, serviceInfo := range g.serviceInfos {
		if serviceInfo.Name == serviceInfoName {
			klog.Infof("Delete aggregator service info %s", serviceInfoName)
			delete(g.serviceInfos, key)
			break
		}
	}
}
