# monarch-kubernetes

Native Kubernetes support for [Monarch](https://github.com/meta-pytorch/monarch).

# Directory structure
operator/ - Contains the operator code that can apply the CRD to a cluster
images/ - Contains the Dockerfile for building the monarch worker image

# How to build


```
make generate
make manifests
# By default IMG=controller:latest so change that if you want to apply a different tag.
make docker-build CONTAINER_TOOL=podman
```

# License
This repo is BSD-3 licensed, as found in the LICENSE file.
