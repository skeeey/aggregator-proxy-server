#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

realpath() {
    [[ $1 = /* ]] && echo "$1" || echo "$PWD/${1#./}"
}

REPO_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")"/..)
BINDIR="${REPO_ROOT}"/output
SC_PKG='github.com/skeeey/aggregator-proxy-server'

# Generate deep copies
"${BINDIR}"/deepcopy-gen "$@" \
	--v 1 --logtostderr\
	--go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	--input-dirs "${SC_PKG}/pkg/apis/aggregation/v1" \
	--output-file-base zz_generated.deepcopy

# Generate openapi
"${BINDIR}"/openapi-gen "$@" \
	--v 1 --logtostderr \
	--go-header-file "${REPO_ROOT}"/hack/custom-boilerplate.go.txt \
	--input-dirs "${SC_PKG}/pkg/apis/aggregation/v1,k8s.io/apimachinery/pkg/apis/meta/v1" \
	--output-package "${SC_PKG}/pkg/apis/aggregation/openapi" \
	--report-filename ".api_violation.report"
