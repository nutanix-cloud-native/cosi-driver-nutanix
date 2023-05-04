# Nutanix COSI driver for provisioning and consuming Nutanix object storage in Kubernetes
Nutanix cosi driver is Nutanix specific component that receives requests from the COSI sidecar and calls the appropriate APIs to create buckets, manage their lifecycle and manage access to them.

COSI driver supports these operations:
1. Creation/Deletion of buckets
2. Granting/Revoking bucket access to individual users

## Pre-requisites
1. [Install](https://helm.sh/docs/intro/install/) Helm v3.0.0.
2. [Install](https://kubernetes.io/docs/setup/) a Kubernetes cluster.

## Installation and Running on the cluster
Deploy the cosi-driver on the cluster:

Clone this repo, get into the charts directory and run the below command:
```sh
helm install cosi-driver -n cosi-driver-nutanix --create-namespace .
```

## Uninstalling the Chart
To uninstall/delete the cosi-driver-nutanix chart:
```console
helm uninstall cosi-driver -n cosi-driver-nutanix
```
**NOTE**: The CRDs installed via helm will not be deleted from the above command. Those have to manually deleted.

## Configuration

The following table lists the configurable parameters of the cosi-driver-nutanix chart and their default values.

| Parameter                                          | Description                                                        | Default                                                                      |
|----------------------------------------------------|--------------------------------------------------------------------|------------------------------------------------------------------------------|
| `replicas`                                         | Number of replicas of the objectstorage-provisioner deployment     | `1`                                                                          |
| `nameOverride`                                     | To override the name of the cosi-driver chart                      | `""`                                                                         |
| `fullnameOverride`                                 | To override the full name of the cosi-driver chart                 | `""`                                                                         |
| `image.registry`                                   | Image registry for cosi-driver-nutanix sidecar                     | `ghcr.io/`                                                                   |
| `image.repository`                                 | Image repository for cosi-driver-nutanix sidecar                   | `nutanix-cloud-native/cosi-driver-nutanix`                                   |
| `image.tag`                                        | Image tag for cosi-driver-nutanix sidecar                          | `""`                                                                         |
| `image.pullPolicy`                                 | Image registry for cosi-driver-nutanix sidecar                     | `IfNotPresent`                                                               |
| `secret.endpoint`                                  | Nutanix Object Store instance endpoint                             | `""`                                                                         |
| `secret.access_key`                                | Admin IAM Access key to be used for Nutanix Objects                | `""`                                                                         |
| `secret.secret_key`                                | Admin IAM Secret key to be used for Nutanix Objects                | `""`                                                                         |
| `secret.pc_secret`                                 | PC Credentials in format <prism-ip>:<prism-port>:<user>:<password> | `""`                                                                         |
| `secret.account_name`                              | Account Name is a displayName identifier Prefix for Nutanix        | `"ntnx-cosi-iam-user"`                                                       |
| `cosiController.replicas`                          | Number of replicas of the COSI central controller deployment       | `1`                                                                          |
| `cosiController.logLevel`                          | Verbosity of logs for COSI central controller deployment           | `5`                                                                          |
| `cosiController.image.registery`                   | Image registry for COSI central controller deployment              | `gcr.io/`                                                                    |
| `cosiController.image.repository`                  | Image repository for COSI central controller deployment            | `k8s-staging-sig-storage/objectstorage-controller`                           |
| `cosiController.image.tag`                         | Image tag for COSI central controller deployment                   | `v20221027-v0.1.1-8-g300019f`                                                |
| `cosiController.image.pullPolicy`                  | Image pull policy for COSI central controller deployment           | `Always`                                                                     |
| `objectstorageProvisionerSidecar.logLevel`         | Verbosity of logs for COSI sidecar                                 | `5`                                                                          |
| `objectstorageProvisionerSidecar.image.registery`  | Image registry for COSI sidecar                                    | `gcr.io/`                                                                    |
| `objectstorageProvisionerSidecar.image.repository` | Image repository for COSI sidecar                                  | `k8s-staging-sig-storage/objectstorage-sidecar/objectstorage-sidecar@sha256` |
| `objectstorageProvisionerSidecar.image.tag`        | Image tag for COSI sidecar                                         | `589c0ad4ef5d0855fe487440e634d01315bc3d883f91c44cb72577ea6e12c890`           |
| `objectstorageProvisionerSidecar.image.pullPolicy` | Image pull policy for COSI sidecar                                 | `Always`                                                                     |


### Configuration examples:

Install the driver in the `cosi-driver-nutanix` namespace (add the `--create-namespace` flag if the namespace does not exist):

```console
helm install cosi-driver -n cosi-driver-nutanix
```

Individual configurations can be set by using `--set key=value[,key=value]` like:
```console
helm install cosi-driver -n cosi-driver-nutanix . --set replicas=2 
```
In the above command `replicas` refers to one of the variables defined in the values.yaml file.

All the options can also be specified in a value.yaml file:

```console
helm install cosi-driver -n cosi-driver-nutanix -f value.yaml .
```
---

## Support
### Community Plus

This code is developed in the open with input from the community through issues and PRs. A Nutanix engineering team serves as the maintainer. Documentation is available in the project repository.

Issues and enhancement requests can be submitted in the [Issues tab of this repository](https://github.com/nutanix-cloud-native/cosi-driver-nutanix/issues). Please search for and review the existing open issues before submitting a new issue.

## License

Copyright 2021-2022 Nutanix, Inc.

The project is released under version 2.0 of the [Apache license](http://www.apache.org/licenses/LICENSE-2.0).
