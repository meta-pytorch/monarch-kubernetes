# monarch-kubernetes

Native Kubernetes support for [Monarch](https://github.com/meta-pytorch/monarch).

# Directory structure
operator/ - Contains the operator code that can apply the CRD to a cluster
images/ - Contains the Dockerfile for building the monarch worker image

# How to build


```
IMG=<your image here>
make generate
make manifests
make docker-build IMG=$IMG CONTAINER_TOOL=podman
```

# License
This repo is BSD-3 licensed, as found in the LICENSE file.
