### Create and query a configmap

```sh
# create a configmap
curl -v \
    -X POST \
    -d @- \
    -H 'Content-Type: application/json' \
    http://localhost:8001/apis/aggregation.open-cluster-management.io/v1/clusterstatuses/spokecluster1/aggregator/v1/namespaces/default/configmaps <<'EOF'
{
  "apiVersion": "v1",
  "kind": "ConfigMap",
  "metadata": {
      "name": "mytestcm"
  },
  "data": {
      "test-key": "test-val"
  }
}
EOF

# query the configmap
curl -v http://localhost:8001/apis/aggregation.open-cluster-management.io/v1/clusterstatuses/spokecluster1/aggregator/v1/namespaces/default/configmaps/mytestcm
```