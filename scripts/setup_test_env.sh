#!/bin/bash

usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -t, --use_triton           Use this flag to deploy triton image to be used as objectstore."
    echo "  -n, --namespace NAMESAPCE      CLuster namespace for the COSI deployment [default = cosi]"
    echo "  -o, --oss_endpoint ENDPOINT      Nutanix Object Store instance endpoint, eg. "http://10.51.142.82:80"."
    echo "  -i, --pc_endpoint ENDPOINT      Prism Central endpoint, eg. "https://10.51.142.82:9440"."
    echo "  -u, --pc_user USERNAME      Prism Central username. [default = admin]"
    echo "  -p, --pc_pass PASSWORD      Prism Central password."
    echo "  -a, --access_key KEY      Admin IAM Access key to be used for Nutanix Objects."
    echo "  -s, --secret_key KEY      Admin IAM Secret key to be used for Nutanix Objects."
    echo "  -h, --help                Display this help and exit."
    echo ""
    exit 1
}

USE_TRITON=""
NODE_IP=""
DRIVER_NAMESPACE="cosi"
OSS_ENDPOINT=""
PC_ENDPOINT=""
PC_USERNAME="admin"
PC_PASSWORD=""
ACCESS_KEY=""
SECRET_KEY=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -t|--use_triton)
            USE_TRITON="true"
            shift 1
            ;;
        -n|--namespace)
            DRIVER_NAMESPACE="$2"
            shift 2
            ;;
        -o|--oss_endpoint)
            OSS_ENDPOINT="$2"
            shift 2
            ;;
        -i|--pc_endpoint)
            PC_ENDPOINT="$2"
            shift 2
            ;;
        -u|--pc_user)
            PC_USERNAME="$2"
            shift 2
            ;;
        -p|--pc_pass)
            PC_PASSWORD="$2"
            shift 2
            ;;
        -a|--access_key)
            ACCESS_KEY="$2"
            shift 2
            ;;
        -s|--secret_key)
            SECRET_KEY="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "[ERROR] Unknown option: $1"
            usage
            ;;
    esac
done

if [[ -z $USE_TRITON ]]; then
    if [[ -z "$OSS_ENDPOINT" ]]; then
        echo "[ERROR] --oss_endpoint is required."
        usage
    fi

    if [[ -z "$PC_ENDPOINT" ]]; then
        echo "[ERROR] --pc_endpoint is required."
        usage
    fi

    if [[ -z "$PC_PASSWORD" ]]; then
        echo "[ERROR] --pc_pass is required."
        usage
    fi

    if [[ -z "$ACCESS_KEY" ]]; then
        echo "[ERROR] --access_key is required."
        usage
    fi

    if [[ -z "$SECRET_KEY" ]]; then
        echo "[ERROR] --secret_key is required."
        usage
    fi
fi

echo "[INFO] Verifying Kubernetes cluster."
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo "[ERROR] Unable to connect to Kubernetes cluster."
    exit 1
fi
echo "[INFO] Cluster is running."

kubectl config view > /tmp/cosi-kubeconfig.yaml
echo "[INFO] kubeconfig file stored at '/tmp/cosi-kubeconfig.yaml'"

echo "[INFO] Removing finalizers from CRs"
for resource in $(kubectl get secret -n="${DRIVER_NAMESPACE}" -o=jsonpath='{.items[*].metadata.name}' 2> /dev/null);
do
    kubectl patch secret -n="${DRIVER_NAMESPACE}" "${resource}" -p='{"metadata":{"finalizers":null}}' --type=merge > /dev/null 2>&1
done

for resource in $(kubectl get bucketclaim.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" -o=jsonpath='{.items[*].metadata.name}' 2> /dev/null);
do
    kubectl patch bucketclaim.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" "${resource}" -p='{"metadata":{"finalizers":null}}' --type=merge > /dev/null 2>&1
done

for resource in $(kubectl get bucketaccess.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" -o=jsonpath='{.items[*].metadata.name}' 2> /dev/null);
do
    kubectl patch bucketaccess.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" "${resource}" -p='{"metadata":{"finalizers":null}}' --type=merge > /dev/null 2>&1
done

for resource in $(kubectl get bucket.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" -o=jsonpath='{.items[*].metadata.name}' 2> /dev/null);
do
    kubectl patch bucket.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" "${resource}" -p='{"metadata":{"finalizers":null}}' --type=merge > /dev/null 2>&1
done

for resource in $(kubectl get bucketaccessclass.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" -o=jsonpath='{.items[*].metadata.name}' 2> /dev/null);
do
    kubectl patch bucketaccessclass.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" "${resource}" -p='{"metadata":{"finalizers":null}}' --type=merge > /dev/null 2>&1
done

for resource in $(kubectl get bucketclass.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" -o=jsonpath='{.items[*].metadata.name}' 2> /dev/null);
do
    kubectl patch bucketclass.objectstorage.k8s.io -n="${DRIVER_NAMESPACE}" "${resource}" -p='{"metadata":{"finalizers":null}}' --type=merge > /dev/null 2>&1
done

if kubectl get namespace "$DRIVER_NAMESPACE" > /dev/null 2>&1; then
    echo "[INFO] Cleaning namespace: $DRIVER_NAMESPACE"
    kubectl delete ns $DRIVER_NAMESPACE > /dev/null 2>&1
fi

echo "[INFO] Creating namespace: $DRIVER_NAMESPACE"
kubectl create ns $DRIVER_NAMESPACE > /dev/null 2>&1

echo "[INFO] Cleaning CRs in default namespace"
kubectl delete bucketclasses.objectstorage.k8s.io --all > /dev/null 2>&1
kubectl delete bucketaccessclasses.objectstorage.k8s.io --all > /dev/null 2>&1
kubectl delete buckets.objectstorage.k8s.io --all > /dev/null 2>&1

if [[ -n $USE_TRITON ]]; then
    echo "[INFO] Deploying triton on cluster"
    kubectl apply -f ./project/resources/triton.yaml -n "${DRIVER_NAMESPACE}" > /dev/null 2>&1

    NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
    OSS_ENDPOINT="http://objects-triton-svc:80"
    ACCESS_KEY="Nutanix"
    SECRET_KEY="Nutanix"
    PC_ENDPOINT="http://objects-triton-svc:5556"
    PC_PASSWORD="password"
fi

echo "[INFO] Installing the COSI Helm chart from ./charts"
helm install cosi-driver -n "${DRIVER_NAMESPACE}" ./charts/ \
    --set=image.tag=latest \
    --set=secret.endpoint="${OSS_ENDPOINT}" \
    --set=secret.access_key="${ACCESS_KEY}" \
    --set=secret.secret_key="${SECRET_KEY}" \
    --set=secret.pc_endpoint="${PC_ENDPOINT}" \
    --set=secret.pc_username="${PC_USERNAME}" \
    --set=secret.pc_password="${PC_PASSWORD}" \
    --set=tls.s3.insecure=true \
    --set=tls.pc.insecure=true > /dev/null 2>&1

echo "[INFO] Waiting for deployment to be available."
kubectl wait --for=condition=available --timeout=60s --namespace="${DRIVER_NAMESPACE}" deployments objectstorage-provisioner > /dev/null 2>&1

echo "[INFO] Exporting environment variables"
export KUBECONFIG="/tmp/cosi-kubeconfig.yaml"
export DRIVER_NAMESPACE="${DRIVER_NAMESPACE}"
export OSS_ENDPOINT="${OSS_ENDPOINT}"
export PC_ENDPOINT="${PC_ENDPOINT}"
export PC_USERNAME="${PC_USERNAME}"
export PC_PASSWORD="${PC_PASSWORD}"
export ACCESS_KEY="${ACCESS_KEY}"
export SECRET_KEY="${SECRET_KEY}"
export NODE_IP="${NODE_IP}"
export USE_TRITON="${USE_TRITON}"

echo "[INFO] Test environment is ready!"

echo "[INFO] Starting E2E tests"
make e2e-tests

echo "[INFO] E2E Tests completed"

echo "[INFO] Cleaning up:"
echo "[INFO] Uninstalling COSI Driver"
helm uninstall cosi-driver -n "${DRIVER_NAMESPACE}" > /dev/null 2>&1
if [[ -n $USE_TRITON ]]; then
    echo "[INFO] Deleting triton from cluster"
    kubectl delete -f ./project/resources/triton.yaml -n "${DRIVER_NAMESPACE}" > /dev/null 2>&1
fi
echo "[INFO] Deleting /tmp/cosi-kubeconfig.yaml file"
rm -f /tmp/cosi-kubeconfig.yaml > /dev/null 2>&1