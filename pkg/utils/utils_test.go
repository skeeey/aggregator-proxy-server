package utils

import "testing"

func TestGetSubResource(t *testing.T) {
	if _, err := GetSubResource(""); err == nil {
		t.Errorf("Expect empty error, but failed")
	}

	wrongPath := "/apis/aggregation.open-cluster-management.io/v1/clusterstatuses/aggregator/"
	if _, err := GetSubResource(wrongPath); err == nil {
		t.Errorf("Expect format error, but failed")
	}

	pathWithoutOptions := "/apis/aggregation.open-cluster-management.io/v1/clusterstatuses/aggregator/test/subres"
	subResource, err := GetSubResource(pathWithoutOptions)
	if err != nil {
		t.Errorf("Expect no error, but failed, %v", err)
	}
	if subResource != "subres" {
		t.Errorf("Expect subres, but %s", subResource)
	}

	pathWithOptions := "/apis/aggregation.open-cluster-management.io/v1/clusterstatuses/test/aggregator/subres2/test"
	subResource, err = GetSubResource(pathWithOptions)
	if err != nil {
		t.Errorf("Expect no error, but failed, %v", err)
	}
	if subResource != "subres2" {
		t.Errorf("Expect subres, but %s", subResource)
	}
}
