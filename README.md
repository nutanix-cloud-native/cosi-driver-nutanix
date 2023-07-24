# COSI Driver Nutanix
Nutanix COSI Driver provides reference implementation for Container Object Storage Interface (COSI) API for Nutanix Object Store

## Install CRDs
```sh
$ git clone github.com/kubernetes-sigs/container-object-storage-interface-api
$ cd container-object-storage-interface-api
$ git checkout 2504944fc33162a34a8a95d6f935cf35c4d08762
$ kubectl create -k .
```
## Install COSI controller
```sh
$ git clone github.com/kubernetes-sigs/container-object-storage-interface-controller
$ cd container-object-storage-interface-controller
$ git checkout 5240fb3aceded346058bdae116e39fabac8897aa
$ kubectl create -k .
```

Following pods will execute in the default namespace :
```sh
NAME                                        READY   STATUS    RESTARTS   AGE
objectstorage-controller-6fc5f89444-4ws72   1/1     Running   0          2d6h
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

## Create Bucket Claim 
```sh
$ kubectl create -f project/examples/bucketclass.yaml
$ kubectl create -f project/examples/bucketclaim.yaml
```
A new bucket is created on nutanix object store using Object Store credentials (`secret.yaml`) and a Bucket(B) custom resource gets created.
```sh
$ kubectl get bucket
NAME                                                      AGE
sample-bucketclass-ed073779-329e-4aff-b7f8-f5bdd54e06d5   7s
```
## Grant Bucket Access
```sh
$ kubectl create -f project/examples/bucketaccessclass.yaml
$ kubectl create -f project/examples/bucketaccess.yaml
$ kubectl get bucketaccess
NAME                                  AGE
sample-bucketaccess                   5s
```
A new Nutanix Object Store user (userName of the format <account-name>_ba-<bucketaccess-UUID>) is created using PC credentials (`secret.yaml`) and the newly created bucket is shared with this new user

## Consuming the bucket in an app
In the app, `bucketaccess` can be consumed as a volume mount. A secret is created with the name provided in the bucketaccess spec field `credentialsSecretName` which can be mounted onto to the pod:
```sh
$ kubectl get secret
NAME          TYPE      DATA   AGE
bucketcreds   Opaque     1     24h
```

```yaml
spec:
  containers:
      volumeMounts:
        - name: cosi-secrets
          mountPath: /data/cosi
  volumes:
  - name: cosi-secrets
    secret:
      secretName: bucketcreds
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

## Deletion of newly created user
```
$ kubectl delete bucketaccess sample-bucketaccess
```

## Deletion of newly created bucket
```
$ kubectl delete bucketclaim sample-bucketclaim
```

## Updating the Nutanix Object Store config
Update the `objectstorage-provisioner` secret that is used by the provisioner deployment with the new config
```
  # Nutanix Object Store instance endpoint, eg. "http://10.51.142.82:80"
  ENDPOINT: "http://10.51.155.148:80"
  # Admin IAM Access key to be used for Nutanix Objects
  ACCESS_KEY: ""
  # Admin IAM Secret key to be used for Nutanix Objects
  SECRET_KEY: ""
  # PC Credentials in format <prism-ip>:<prism-port>:<user>:<password>. 
  # eg. "<ip>:<port>:user:password"
  PC_SECRET: ""
```

Then restart the provisioner pod so that the new secret changes getting mounted on the new pod and will thereon be used.
```
$ kubectl -n ntnx-system get pods
NAME                                         READY   STATUS    RESTARTS   AGE
objectstorage-provisioner-6c8df56cc6-lqr26   2/2     Running   0          26h
```

```
$ kubectl delete pod objectstorage-provisioner-6c8df56cc6-lqr26 -n ntnx-system
```
New pod comes up which will be having the updated config
```
$ kubectl -n ntnx-system get pods
NAME                                         READY   STATUS    RESTARTS   AGE
objectstorage-provisioner-5f3we89tt2-tfy357   2/2     Running   0          2s
```