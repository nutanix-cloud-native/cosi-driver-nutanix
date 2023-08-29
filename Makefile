# Copyright 2022 Nutanix Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

CMDS=cosi-driver-nutanix

REGISTRY_NAME=ghcr.io/nutanix-cloud-native/cosi-driver-nutanix
IMAGE_TAG=latest

all: build

.PHONY: build-% build container-% container clean

# A space-separated list of all commands in the repository, must be
# set in main Makefile of a repository.
# CMDS=

# Revision that gets built into each binary via the main.version
# string. Uses the `git describe` output based on the most recent
# version tag with a short revision suffix or, if nothing has been
# tagged yet, just the revision.
REV=$(shell git describe --long --tags --match='v*' --dirty 2>/dev/null || git rev-list -n1 HEAD)

# Specific packages can be excluded from each of the tests below by setting the *_FILTER_CMD variables
# to something like "| grep -v 'github.com/kubernetes-csi/project/pkg/foobar'". See usage below.

build-%:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-X main.version=$(REV) -extldflags "-static"' -o ./bin/$* ./cmd/$*

container-%: build-%
	docker build -t $(REGISTRY_NAME):$(IMAGE_TAG) -f package/docker/Dockerfile --label revision=$(REV) .

build: $(CMDS:%=build-%)
container: $(CMDS:%=container-%)

.PHONY: docker-push
docker-push:
	docker push $(REGISTRY_NAME):$(IMAGE_TAG)

clean:
	-rm -rf bin
