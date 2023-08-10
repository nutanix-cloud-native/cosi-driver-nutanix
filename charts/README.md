# Nutanix COSI driver for provisioning and consuming Nutanix object storage in Kubernetes
Nutanix cosi driver is Nutanix specific component that receives requests from the COSI sidecar and calls the appropriate APIs to create buckets, manage their lifecycle and manage access to them.

COSI driver supports these operations:
1. Creation/Deletion of buckets
2. Granting/Revoking bucket access to individual users

## Pre-requisites
1. [Install](https://helm.sh/docs/intro/install/) Helm v3.0.0.
2. [Install](https://kubernetes.io/docs/setup/) a Kubernetes cluster.

## Installation and running on the cluster
Deploy the cosi-driver on the cluster:

Clone this repo, get into the charts directory and run the below command:
```sh
helm install cosi-driver -n cosi-driver-nutanix --create-namespace .
```

## Uninstalling the chart
To uninstall/delete the cosi-driver-nutanix chart:
```console
helm uninstall cosi-driver -n cosi-driver-nutanix
```
**NOTE**: The CRDs installed via helm will not be deleted from the above command. Those have to manually deleted.

## Upgrading the chart
To upgrade the cosi-driver-nutanix chart to a a particular version can be achieved via the helm upgrade command with the following syntax:
helm upgrade [RELEASE] [CHART] [flags]

Example:
```console
helm upgrade cosi-driver -n cosi-driver-nutanix .
```

To know more about the various flag options used with upgrade command check out the [helm_upgrade]("https://helm.sh/docs/helm/helm_upgrade/") official document.

## Configuration

The following table lists the configurable parameters of the cosi-driver-nutanix chart and their default values.

| Parameter                                          | Description                                                        | Default                                                                      |
|----------------------------------------------------|--------------------------------------------------------------------|------------------------------------------------------------------------------|
| `nameOverride`                                     | To override the name of the cosi-driver chart                      | `""`                                                                         |
| `fullnameOverride`                                 | To override the full name of the cosi-driver chart                 | `""`                                                                         |
| `image.registry`                                   | Image registry for cosi-driver-nutanix sidecar                     | `ghcr.io/`                                                                   |
| `image.repository`                                 | Image repository for cosi-driver-nutanix sidecar                   | `nutanix-cloud-native/cosi-driver-nutanix`                                   |
| `image.tag`                                        | Image tag for cosi-driver-nutanix sidecar                          | `""`                                                                         |
| `image.pullPolicy`                                 | Image registry for cosi-driver-nutanix sidecar                     | `IfNotPresent`                                                               |
| `secret.enabled`                                   | Enables K8s secret deployment for Nutanix Object Store             | `true`                                                                       |
| `secret.endpoint`                                  | Nutanix Object Store instance endpoint                             | `""`                                                                         |
| `secret.access_key`                                | Admin IAM Access key to be used for Nutanix Objects                | `""`                                                                         |
| `secret.secret_key`                                | Admin IAM Secret key to be used for Nutanix Objects                | `""`                                                                         |
| `secret.pc_secret`                                 | PC Credentials in format <prism-ip>:<prism-port>:<user>:<password> | `""`                                                                         |
| `secret.account_name`                              | Account Name is a displayName identifier Prefix for Nutanix        | `"ntnx-cosi-iam-user"`                                                       |
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
 helm install cosi-driver -n cosi-driver-nutanix .
 ```

 Individual configurations can be set by using `--set key=value[,key=value]` like:
 ```console
 helm install cosi-driver -n cosi-driver-nutanix . --set cosiController.logLevel=2
 ```
 In the above command `cosiController.logLevel` refers to one of the variables defined in the values.yaml file.

 All the options can also be specified in a values.yaml file:

 ```console
 helm install cosi-driver -n cosi-driver-nutanix -f values.yaml .
 ```

### Steps to update the Nutanix Object store details while installing COSI:
1. Open Prism Central UI in any browser and go the objects page. Below I already have a object store called `cosi` deployed ready for use. On the right side of the object store, you will see the objects Public IPs which you can use as the endpoint in the format: `http:<objects public ip>:80`.
<img width="1512" alt="Screenshot 2023-08-10 at 4 31 41 PM" src="https://github.com/nutanix-core/k8s-juno/assets/44068648/121ae44f-cb63-452a-a772-cca5ff428cf4">

2. On the side navigation bar click the `Access Keys` tab and then click on `Add People`.
<img width="1510" alt="Screenshot 2023-08-10 at 4 41 41 PM" src="https://github.com/nutanix-core/k8s-juno/assets/44068648/a293b529-e9c1-495c-9f96-51f2c4fdf92c">

3. Add a new email address and name and click `Next`.
<img width="502" alt="Screenshot 2023-08-10 at 4 42 41 PM" src="https://github.com/nutanix-core/k8s-juno/assets/44068648/72494b72-efe9-42c3-8f3c-40f9563aa041">

4. Now click thr `Generate Keys` button.
<img width="496" alt="Screenshot 2023-08-10 at 4 43 00 PM" src="https://github.com/nutanix-core/k8s-juno/assets/44068648/ed099ee3-cd9d-4025-8052-ddb6ac83eae0">

5. After the keys are generated downlaod the generated keys.
<img width="494" alt="Screenshot 2023-08-10 at 4 43 16 PM" src="https://github.com/nutanix-core/k8s-juno/assets/44068648/876e3f6d-7faa-4106-88a5-9117ba2424f1">

6. Now, in the `Access Key` tab you will be able to see the person you just added.
<img width="1512" alt="Screenshot 2023-08-10 at 4 43 52 PM" src="https://github.com/nutanix-core/k8s-juno/assets/44068648/33e1a29a-a6bd-41e5-8bfa-2eb6aafb1e4b">

7. The keys file that you downloaded will be a text file which will contain the `Access Key` and `Secret Key` that you need to update in helm values.

Keep the endpoint details collected in step#1, the keys details mentioned in step#7 and Nutanix Prism-Central(PC) details in the format `<prism-ip>:<prism-port>:<user>:<password>` handy.

To update the secret details use the below helm command to deploy Nutanix cosi driver:
```console
helm install cosi-driver -n cosi-driver-nutanix . --set secret.enabled=true --set secret.endpoint=<object-store-endpoint> secret.access_key=<object-store-access-key> --set secret.secret_key=<object-store-secret-key>  --set secret.pc_secret=<nutanix-pc-details>
```
 ---

## Support
### Community Plus

This code is developed in the open with input from the community through issues and PRs. A Nutanix engineering team serves as the maintainer. Documentation is available in the project repository.

Issues and enhancement requests can be submitted in the [Issues tab of this repository](https://github.com/nutanix-cloud-native/cosi-driver-nutanix/issues). Please search for and review the existing open issues before submitting a new issue.

## License

Copyright 2021-2022 Nutanix, Inc.

The project is released under version 2.0 of the [Apache license](http://www.apache.org/licenses/LICENSE-2.0).
