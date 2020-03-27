package utils

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

//MatchLabelForLabelSelector match labels for labelselector, if labelSelecor is nil, select everything
func MatchLabelForLabelSelector(targetLabels map[string]string, labelSelector *metav1.LabelSelector) bool {
	selector, err := convertLabels(labelSelector)
	if err != nil {
		return false
	}
	if selector.Matches(labels.Set(targetLabels)) {
		return true
	}
	return false
}

// GetSubResource returns the sub-resource behind aggregator in a request path.
// request path: /apis/<group>/<version>/clusterstatuses/<cluster-name>/aggregator/<sub-resource>/xxx
func GetSubResource(requestPath string) (string, error) {
	requestPath = strings.Trim(requestPath, "/")
	if requestPath == "" {
		return "", fmt.Errorf("empty path")
	}

	pathParts := strings.Split(requestPath, "/")
	if len(pathParts) < 7 {
		return "", fmt.Errorf("wrong path format")
	}

	//TODO: may need to check each field name
	return pathParts[6], nil
}

func convertLabels(labelSelector *metav1.LabelSelector) (labels.Selector, error) {
	if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return labels.Nothing(), err
		}

		return selector, nil
	}

	return labels.Everything(), nil
}
