BINDIR ?= output

generate_files:
	hack/update-apiserver-gen.sh

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -o $(BINDIR)/aggregator-proxy-server  cmd/proxy-server/proxyserver.go

images: clean build
	docker build . -f Dockerfile -t aggregator-proxy-server:0.0.1

clean::
	rm -rf $(BINDIR)/aggregator-proxy
