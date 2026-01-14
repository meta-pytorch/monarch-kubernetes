# monarch-kubernetes

Contains a reference CRD and operator for [Monarch](https://github.com/meta-pytorch/monarch) workers.

This can be used to deploy Monarch workers to a Kubernetes cluster. The operator applies the CRD
to the cluster and applies labels so the controller can discover them.

# Current status

We have a CRD and an operator. This can be used to provision workers on the kubernetes cluster.
[KubernetesJob](https://github.com/meta-pytorch/monarch/blob/main/python/monarch/_src/job/kubernetes.py)
can then be used to connect and create a proc mesh on the workers.

# Directory structure
operator/ - Contains the operator code that can apply the CRD to a cluster

images/ - Contains the Dockerfile for building the monarch worker image

# How to build
```
cd operator
make generate
make manifests
# By default IMG=controller:latest so change that if you want to apply a different tag.
make docker-build CONTAINER_TOOL=podman
```

# How to run
```
cd operator
# Run controller locally
make run
# By default IMG=controller:latest so change that if you want to apply a different tag.
# Or, deploy controller to cluster
make deploy
```

# How to test
```
cd operator
# Test controller
make test
```

# Examples
Check out [hello_kubernetes_job.py](https://github.com/meta-pytorch/monarch/blob/main/examples/kubernetes/hello_kubernetes_job.py) for an example of how to use the KubernetesJob class.

# Common issues
1. Ensure you have same version of Monarch installed on the workers as the controller. Monarch doesn't provide forward/backward compatibility for the controller/worker protocol.

# License
This repo is BSD-3 licensed, as found in the LICENSE file.
