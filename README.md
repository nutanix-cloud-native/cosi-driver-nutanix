# COSI Driver Nutanix
Nutanix COSI Driver provides reference implementation for Container Object Storage Interface (COSI) API for Nutanix Object Store

## Install CRDs
```sh
$ git clone github.com/kubernetes-sigs/container-object-storage-interface-api

$ cd container-object-storage-interface-api

$ git checkout 18f05e92c24b97b10b95319449b30cce2c0b3538

$ kubectl create -k .
```
## Install COSI controller
```sh
$ git clone github.com/kubernetes-sigs/container-object-storage-interface-controller

$ cd container-object-storage-interface-controller

$ git checkout a23ec2beea89e1db2e7b0bdc4e518edbbae34df6

$ kubectl create -k .
```
## Install Node adapter
```sh
$ kubectl create -k github.com/kubernetes-sigs/container-object-storage-interface-csi-adapter
```

Following pods will execute in the default namespace :
```sh
NAME                                        READY   STATUS    RESTARTS   AGE
objectstorage-controller-6fc5f89444-4ws72   1/1     Running   0          2d6h
objectstorage-csi-adapter-wsl4l             3/3     Running   0          2d6h
```

## Building, Installing, Setting Up
Code can be compiled using:
```sh
$ cd k8s-ntnx-object-cosi
$ make build
```
### Update the following credentials in project/resources/secret.yaml 

- `ENDPOINT` : Nutanix Object Store Endpoint
- `ACCESS_KEY` : Nutanix Object Store Access Key
- `SECRET_KEY` : Nutanix Object Store Secret Key
- `PC_SECRET` : Prism Central Credentials in the form 'prism-ip:prism-port:username:password'
- `ACCOUNT_NAME` (Optional) : DisplayName identifier prefix for Nutanix Object Store (Default_Prefix: ntnx-cosi-iam-user)

Build docker image and provide tag as `ntnx/ntnx-cosi-driver:latest`
```sh
$ make container
$ docker tag ntnx-cosi-driver:latest ntnx/ntnx-cosi-driver:latest
```
Now start the sidecar and cosi driver with:
```sh
$ kubectl apply -k project/.
$ kubectl -n ntnx-system get pods
NAME                                         READY   STATUS    RESTARTS   AGE
objectstorage-provisioner-6c8df56cc6-lqr26   2/2     Running   0          26h
```

## Create Bucket Request
```sh
$ kubectl create -f project/examples/bucketclass.yaml
$ kubectl create -f project/examples/bucketrequest.yaml
```
A new bucket is created on nutanix object store using Object Store credentials (`secret.yaml`) and a Bucket(B) custom resource gets created
```sh
$ kubectl get bucket
NAME                                        AGE
test-ed073779-329e-4aff-b7f8-f5bdd54e06d5   7s
```
## Grant Bucket Access
```sh
$ kubectl create -f project/examples/configmap.yaml
$ kubectl create -f project/examples/bucketaccessclass.yaml
$ kubectl create -f project/examples/bucketaccessrequest.yaml
```
A new Nutanix Object Store user (userName corresponding to BucketAccessRequest) is created using PC credentials (`secret.yaml`) and the newly created bucket is shared with this new user

```sh
$ kubectl get bucketaccess
NAME                                   AGE
39233394-4e37-4842-bbdd-c97edc7e9483   71s
```
## Consuming the bucket in an app
In the app, `bucketaccessrequest(bar)` can be consumed as a volume mount:
```yaml
spec:
  containers:
      volumeMounts:
        - name: cosi-secrets
          mountPath: /data/cosi
  volumes:
  - name: cosi-secrets
    csi:
      driver: objectstorage.k8s.io
      volumeAttributes:
        bar-name: sample-bar
        bar-namespace: default
```
An example awscli pod can be found at `project/examples/awscliapppod.yaml`

```
$ kubectl create -f project/examples/awscliapppod.yaml
$ kubectl exec -it awscli bash
root@awscli:/aws# sh test.sh
```
`test.sh` is a sample test script in awscli pod which will do a list-buckets and put-object operation using credentials of newly created user. 

Credentials are available at `/data/cosi/credentials`
in the awscli pod

## Limitations

- Deletion of BucketRequest(BR) and BucketAccessRequest(BAR) not supported. Presently it needs to be done manually