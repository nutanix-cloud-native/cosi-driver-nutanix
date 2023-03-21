#!/bin/bash
set +x
PKG_DIR=github.com/nutanix-core
PKG=${PKG_DIR}/k8s-ntnx-object-cosi

mkdir -p $GOPATH/{src,bin,pkg}

# Move repo folder to within GOPATH
mkdir -p $GOPATH/src/${PKG_DIR}
cp -a ~/project/. $GOPATH/src/${PKG}

# Setup git so as to use https with basic auth instead of https
git config --global url."git@github.com:".insteadOf "https://github.com/"

cd $GOPATH/src/$PKG