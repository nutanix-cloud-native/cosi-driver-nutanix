#!/bin/bash
set -x
set -e
PKG_DIR=github.com/nutanix-core
PKG=${PKG_DIR}/k8s-ntnx-object-cosi

mkdir -p ~/project/package/docker/bin
go build -o ~/project/package/docker/bin/ntnx-system $PKG/cmd/ntnx-system
