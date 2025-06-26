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
- `PC_ENDPOINT` : Prism Central endpoint'
- `PC_SECRET` : Prism Central Credentials in the form 'username:password'
- `S3_INSECURE` : Controls whether certificate chain will be validated for S3 endpoint (Default: "false")
- `PC_INSECURE` : Controls whether certificate chain will be validated for Prism Central (Default: "false")
- `ACCOUNT_NAME` (Optional) : DisplayName identifier prefix for Nutanix Object Store (Default_Prefix: ntnx-cosi-iam-user)
- `S3_CA_CERT` (Optional) : Base64 encoded content of the root certificate authority file for S3 endpoint (Default: "")
- `PC_CA_CERT` (Optional) : Base64 encoded content of the root certificate authority file for Prism Central (Default: "")

**NOTE**: Certificates should be in `PEM` encoded format.

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
This Pod will list the buckets and then writes a test file to the new bucket.
```sh
$ kubectl create -f project/examples/awscliapppod.yaml
$ kubectl logs awscli
Defaulted container "awscli" out of: awscli, write-aws-credentials (init), write-test-file (init)
+ aws s3 ls
2024-12-20 19:38:40 sample-bucketclassc949a8c0-4c73-46ea-ace8-20071bff8102
++ cat /tmp/test-directory/file.txt
+ readonly BUCKET_NAME=sample-bucketclassc949a8c0-4c73-46ea-ace8-20071bff8102
+ BUCKET_NAME=sample-bucketclassc949a8c0-4c73-46ea-ace8-20071bff8102
++ date +%Y%m%d_%H%M%S
+ readonly FILE_NAME=20241220_213034.txt
+ FILE_NAME=20241220_213034.txt
+ aws s3 cp /tmp/test-directory/file.txt s3://sample-bucketclassc949a8c0-4c73-46ea-ace8-20071bff8102/20241220_213034.txt
upload: ../tmp/test-directory/file.txt to s3://sample-bucketclassc949a8c0-4c73-46ea-ace8-20071bff8102/20241220_213034.txt
+ aws s3 cp s3://sample-bucketclassc949a8c0-4c73-46ea-ace8-20071bff8102/20241220_213034.txt -
sample-bucketclassc949a8c0-4c73-46ea-ace8-20071bff8102
```

Credentials are available at `/data/cosi/BucketInfo` in the awscli Pod.

### Deletion of newly created user
```sh
$ kubectl delete bucketaccess sample-bucketaccess
$ kubectl delete bucketaccessclass sample-bucketaccessclass
```

### Deletion of newly created bucket
```sh
$ kubectl delete bucketclaim sample-bucketclaim
$ kubectl delete bucketclass sample-bucketclass
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
  # Prism Central endpoint, eg. "https://10.51.149.82:9440"
  PC_ENDPOINT: "<ENDPOINT>"
  # PC Credentials in format <user>:<password>. 
  # eg. "user:password"
  PC_SECRET: "<USER>:<PASSWORD>"
  # Controls whether certificate chain will be validated for S3 endpoint
  # If INSECURE is set to true, an insecure connection will be made with
  # the S3 endpoint (Certs will not be used)
  S3_INSECURE: "false"
  # Controls whether certificate chain will be validated for Prism Central
  # If INSECURE is set to true, an insecure connection will be made with
  # the PC endpoint (Certs will not be used)
  PC_INSECURE: "false"
  # Base64 encoded content of the root certificate authority file for S3 endpoint
  # empty if no certs should be used.
  # Example, 
  # S3_CA_CERT: "LS0tLS1CRU...SUZJQ0FURS0tLS0tCg=="
  S3_CA_CERT: ""
  # Base64 encoded content of the root certificate authority file for Prism Central
  # empty if no certs should be used.
  PC_CA_CERT: ""
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

## Running Tests
### Unit Tests
Execute the following to run unit tests:
```sh
go test ./...
```
To generate the coverage report and an HTML page to view the report:
```sh
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```
Open the HTML file in any web vrowser to view the coverage of each file.

### E2E Tests
Execute the following to run E2E tests:
```sh
sh scripts/setup_test_env.sh [flags]
```
```
Options:
-o, --oss_endpoint ENDPOINT    Nutanix Object Store instance endpoint, eg. "http://10.51.142.82:80".
-i, --pc_endpoint ENDPOINT     Prism Central endpoint, eg. "https://10.51.142.82:9440".
-u, --pc_user USERNAME         Prism Central username. [default = admin]
-p, --pc_pass PASSWORD         Prism Central password.
-a, --access_key KEY           Admin IAM Access key to be used for Nutanix Objects.
-s, --secret_key KEY           Admin IAM Secret key to be used for Nutanix Objects.
-n, --namespace NAMESAPCE      Cluster namespace for the COSI deployment [default = cosi]
```
You can also run the E2E tests on Triton in a local environment if a real cluster is not available.
To run on Triton, you need the image [http://uranus.corp.nutanix.com/~ankush.patanwal/objects-triton.tar.gz] and k8s cluster running locally (eg. `minikube`) then execute the script with flag `-t` or `--use_triton`.
```sh
sh scripts/setup_test_env.sh -t
```
NOTE: You will need to load the Triton image to the local cluster. In `minikube` this can be done using:
```sh
minikube image load objects-triton:debug
```
