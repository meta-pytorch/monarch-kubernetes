# Running the demo on Meta's internal Monarch AWS cluster



```shell

# Get the credentials
eval $(cloud aws get-creds meta-monarch-k8s --role SSOAdmin)
aws ecr get-login-password --region us-east-1 | podman login --username AWS --password-stdin 383172230265.dkr.ecr.us-east-1.amazonaws.com

# Build the operator image and push it to the registry.
IMG=383172230265.dkr.ecr.us-east-1.amazonaws.com/monarch-k8s:ahmad-test1

cd ~/fbsource/fbcode/monarch-kubernetes/operator
make docker-build docker-push deploy IMG=$IMG CONTAINER_TOOL=podman DOCKER_BUILD_ARGS="--network=host --build-arg http_proxy=$http_proxy --build-arg https_proxy=$https_proxy --build-arg no_proxy=$no_proxy"

# Rollout the operator controller.
kubectl rollout restart deployment/monarch-operator-controller-manager -n monarch-operator-system
kubectl rollout status deployment/monarch-operator-controller-manager -n monarch-operator-system

# Now apply the CRD to the cluster.
cd ~/fbsource/fbcode/monarch-kubernetes/meta
kubectl apply -f simple_demo.yaml
kubectl get statefulsets,pods -n monarch-tests --show-labels
kubectl delete -f simple_demo.yaml
```
