# COSI Driver Nutanix
Nutanix COSI Driver provides reference implementation for Container Object Storage Interface (COSI) API for Nutanix Object Store

## Deploying COSI Driver

### Using Helm
Checkout the detailed helm chart documentation in [charts directory](/charts/README.md) for installing COSI driver using helm.

### Manual Deployment

#### Install CRDs
```sh
$ git clone github.com/kubernetes-sigs/container-object-storage-interface-api
$ cd container-object-storage-interface-api
$ git checkout 2504944fc33162a34a8a95d6f935cf35c4d08762
$ kubectl create -k .
```

#### Install COSI controller
```sh
$ git clone github.com/kubernetes-sigs/container-object-storage-interface-controller
$ cd container-object-storage-interface-controller
$ git checkout 5240fb3aceded346058bdae116e39fabac8897aa
$ kubectl create -k .
```

Following pods will execute in the default namespace:
```sh
NAME                                        READY   STATUS    RESTARTS   AGE
objectstorage-controller-6fc5f89444-4ws72   1/1     Running   0          2d6h
```

#### Install object storage provisioner sidecar with the Nutanix cosi driver

```sh
$ git clone https://github.com/nutanix-cloud-native/cosi-driver-nutanix
$ cd cosi-driver-nutanix
```

**Update the following credentials in project/resources/secret.yaml:**
- `ENDPOINT` : Nutanix Object Store Endpoint
- `ACCESS_KEY` : Nutanix Object Store Access Key
- `SECRET_KEY` : Nutanix Object Store Secret Key
- `PC_SECRET` : Prism Central Credentials in the form 'prism-ip:prism-port:username:password'
- `ACCOUNT_NAME` (Optional) : DisplayName identifier prefix for Nutanix Object Store (Default_Prefix: ntnx-cosi-iam-user)

**Pre-requisites:**
Already deployed Nutanix object-store

**Steps on how to get the above details:**
1. Open Prism Central UI in any browser and go the objects page. Below I already have a object store called `cosi` deployed ready for use. On the right side of the object store, you will see the objects Public IPs which you can use as the endpoint and update it in the secret.yaml file in the format: `http:<objects public ip>:80`. 
<img width="1512" alt="Screenshot 2023-08-10 at 4 31 41 PM" src="https://github.com/nutanix-cloud-native/cosi-driver-nutanix/assets/44068648/ee0d9ef9-5c5a-4a5a-a0c0-ef2d76db118c">

2. On the side navigation bar click the `Access Keys` tab and then click on `Add People`.
<img width="1510" alt="Screenshot 2023-08-10 at 4 41 41 PM" src="https://github.com/nutanix-cloud-native/cosi-driver-nutanix/assets/44068648/646788d8-d4c4-49fb-abfe-b20c14e8bd7f">

3. Add a new email address and name and click `Next`.
<img width="502" alt="Screenshot 2023-08-10 at 4 42 41 PM" src="https://github.com/nutanix-cloud-native/cosi-driver-nutanix/assets/44068648/7b12652d-26b4-49d2-92f1-cdddc658d1da">

4. Now click the `Generate Keys` button.
<img width="496" alt="Screenshot 2023-08-10 at 4 43 00 PM" src="https://github.com/nutanix-cloud-native/cosi-driver-nutanix/assets/44068648/fed3a458-900e-4e3e-9112-af8f3c23b00c">

5. After the keys are generated download the generated keys.
<img width="494" alt="Screenshot 2023-08-10 at 4 43 16 PM" src="https://github.com/nutanix-cloud-native/cosi-driver-nutanix/assets/44068648/09598ff9-e696-45bb-9bb4-b517f3822c71">

6. Now, in the `Access Key` tab you will be able to see the person you just added.
<img width="1512" alt="Screenshot 2023-08-10 at 4 43 52 PM" src="https://github.com/nutanix-cloud-native/cosi-driver-nutanix/assets/44068648/d333cd1c-f59c-4e4b-845d-a7ec950a82c3">

7. The keys file that you downloaded will be a text file which will contain the `Access Key` and `Secret Key` that you need to update in the above secret.yaml file.

After updating the above file, execute these commands: 
```sh
$ kubectl apply -k project/.
$ kubectl -n ntnx-system get pods
NAME                                         READY   STATUS    RESTARTS   AGE
objectstorage-provisioner-6c8df56cc6-lqr26   2/2     Running   0          26h
```

## Quickstart

### Create Bucket Claim 
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

### Grant Bucket Access
```sh
$ kubectl create -f project/examples/bucketaccessclass.yaml
$ kubectl create -f project/examples/bucketaccess.yaml
$ kubectl get bucketaccess
NAME                                  AGE
sample-bucketaccess                   5s
```
A new Nutanix Object Store user (userName of the format <account-name>_ba-<bucketaccess-UUID>) is created using PC credentials (`secret.yaml`) and the newly created bucket is shared with this new user

### Consuming the bucket in an app
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

### Deletion of newly created user
```
$ kubectl delete bucketaccess sample-bucketaccess
```

### Deletion of newly created bucket
```
$ kubectl delete bucketclaim sample-bucketclaim
```

## Updating the Nutanix Object Store config
Update the `objectstorage-provisioner` secret that is used by the running provisioner deployment with the new config
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

## Building Nutanix cosi driver container image
Code can be compiled using:
```sh
$ git clone https://github.com/nutanix-cloud-native/cosi-driver-nutanix
$ cd cosi-driver-nutanix
$ make build
```

Build and push docker image and for your custom resistry name and image tag 
```sh
$ make REGISTRY_NAME=SampleRegistryUsername/cosi-driver-nutanix IMAGE_TAG=latest container
$ make REGISTRY_NAME=SampleRegistryUsername/cosi-driver-nutanix IMAGE_TAG=latest docker-push
```

Your custom image `SampleRegistry/cosi-driver-nutanix:latest` is now ready to be used.
