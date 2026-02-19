# keycloak-operator

Helm chart for the [Keycloak Kubernetes operator](https://www.keycloak.org/operator/installation).

The upstream project publishes static YAML with hardcoded image references, resource limits, and namespace. This chart makes all of that configurable.

## Install

```bash
helm repo add px3-dev https://px3-dev.github.io/keycloak-operator
helm install keycloak-operator px3-dev/keycloak-operator -n keycloak --create-namespace
```

## Override images

Point the operator and Keycloak server images at a mirror registry:

```bash
helm install keycloak-operator px3-dev/keycloak-operator \
  --set image.repository=my-mirror.example.com/keycloak/keycloak-operator \
  --set keycloakImage.repository=my-mirror.example.com/keycloak/keycloak
```

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `image.repository` | `quay.io/keycloak/keycloak-operator` | Operator image |
| `image.tag` | `""` (appVersion) | Operator image tag |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `keycloakImage.repository` | `quay.io/keycloak/keycloak` | Keycloak server image the operator deploys |
| `keycloakImage.tag` | `""` (appVersion) | Keycloak server image tag |
| `imagePullSecrets` | `[]` | Registry credentials |
| `replicas` | `1` | Operator replica count |
| `resources.requests.cpu` | `300m` | CPU request |
| `resources.requests.memory` | `450Mi` | Memory request |
| `resources.limits.cpu` | `700m` | CPU limit |
| `resources.limits.memory` | `450Mi` | Memory limit |
| `serviceAccount.create` | `true` | Create a ServiceAccount |
| `serviceAccount.name` | `""` | Override ServiceAccount name |
| `service.type` | `ClusterIP` | Service type |
| `service.port` | `80` | Service port |

## How the chart is generated

The `chart/` directory is generated from upstream manifests by a Go tool. This means upgrading to a new Keycloak version is mechanical, not a manual YAML diff.

```
upstream kubernetes.yml ──▶ go run ./cmd/generate ──▶ chart/
```

The generator parses the upstream multi-document YAML, extracts RBAC rules, deployment spec, and service config, then produces Helm templates with proper value overrides. RBAC rules are preserved verbatim from upstream.

## Upgrade to a new upstream version

Requires Go and Helm (managed by [mise](https://mise.jdx.dev)):

```bash
mise run generate 26.5.3
```

This downloads the upstream manifests for the given version, regenerates the chart, and lints it. Review the diff, bump `chart/Chart.yaml` version, and commit.

To do it manually:

```bash
# Download upstream manifests
curl -sfLO https://raw.githubusercontent.com/keycloak/keycloak-k8s-resources/26.5.3/kubernetes/kubernetes.yml
curl -sfLO https://raw.githubusercontent.com/keycloak/keycloak-k8s-resources/26.5.3/kubernetes/keycloaks.k8s.keycloak.org-v1.yml
curl -sfLO https://raw.githubusercontent.com/keycloak/keycloak-k8s-resources/26.5.3/kubernetes/keycloakrealmimports.k8s.keycloak.org-v1.yml

# Regenerate
go run ./cmd/generate \
  --manifest kubernetes.yml \
  --crd keycloaks.k8s.keycloak.org-v1.yml \
  --crd keycloakrealmimports.k8s.keycloak.org-v1.yml \
  --output chart

# Verify
helm lint chart
helm template keycloak-operator chart
```

## License

Apache 2.0
