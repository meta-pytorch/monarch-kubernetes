# monarch-kubernetes

`monarch-kubernetes` provides a Kubernetes Custom Resource Definition (CRD) and operator for MonarchMesh, simplifying the deployment and management of [Monarch](https://github.com/meta-pytorch/monarch) workloads on Kubernetes. The operator reconciles MonarchMesh resources and provisions Monarch workers compatible with [KubernetesJob](https://meta-pytorch.org/monarch/api/monarch.job.html#kubernetesjob).

> ⚠️ **Early Development Warning** monarch-kubernetes is currently in an experimental
> stage. You should expect bugs, incomplete features, and APIs that may change
> in future versions. The project welcomes bugfixes, but to make sure things are
> well coordinated you should discuss any significant change before starting the
> work. It's recommended that you signal your intention to contribute in the
> issue tracker, either by filing a new issue or by claiming an existing one.
## Directory Structure

| Directory   | Description                                           |
|-------------|-------------------------------------------------------|
| `operator/` | Operator source code for reconciling the CRD         |
| `docs/`     | Helm Chart package and documentation index           |

## Installation

### Helm Chart (Recommended)

Install the MonarchMesh CRD and operator using Helm:

```bash
# Add the Helm repository
helm repo add monarch-operator https://meta-pytorch.github.io/monarch-kubernetes

# Update repository cache
helm repo update

# Install MonarchMesh CRD and operator
helm install monarch-operator monarch-operator/monarch-operator \
  --namespace monarch-system \
  --create-namespace
```

To uninstall:

```bash
helm uninstall monarch-operator --namespace monarch-system
```

### Manual Installation

#### Build

```bash
cd operator

# Generate code and manifests
make generate
make manifests

# Build the container image (default: IMG=controller:latest)
make docker-build CONTAINER_TOOL=podman
```

#### Deploy

```bash
cd operator

# Option 1: Run the controller locally
make run

# Option 2: Deploy to the cluster (default: IMG=controller:latest)
make deploy
```

## Usage

For a complete example demonstrating how to use the KubernetesJob class with Monarch, see the [hello_kubernetes_job](https://github.com/meta-pytorch/monarch/tree/main/examples/kubernetes/hello_kubernetes_job) example.

## Testing

```bash
cd operator

# Run unit tests
make test

# Run end-to-end tests (sets up a local cluster)
make test-e2e
```

## Troubleshooting

**Version Mismatch:** Ensure the Monarch version installed on workers matches the controller version. Monarch does not provide forward or backward compatibility for the controller/worker protocol.

## License

This project is licensed under the BSD-3-Clause License. See the [LICENSE](LICENSE) file for details.
