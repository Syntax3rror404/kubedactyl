# Kubedactyl Helm chart

Installs the Kubedactyl panel: service account with the required cluster permissions, a single-replica
Deployment (non-root, read-only root filesystem), a Service and optionally an Ingress or a Gateway API
HTTPRoute. The panel installs and updates its own CRDs on start, so the chart contains none.

```bash
helm install kubedactyl oci://ghcr.io/syntax3rror404/charts/kubedactyl --version 0.2.62 \
  -n kubedactyl --create-namespace
kubectl -n kubedactyl logs deploy/kubedactyl | grep setup   # one-time link for the first administrator
```

## Values

| Key | Default | Meaning |
|-----|---------|---------|
| `image.repository` / `image.tag` | `ghcr.io/syntax3rror404/kubedactyl` / appVersion | panel image (linux/amd64) |
| `admin.password` | - | creates `admin` directly instead of the setup page |
| `admin.setupToken` | - | fixed setup token instead of a generated one |
| `admin.existingSecret` | - | secret with `admin-password` and/or `setup-token` |
| `panel.storageClass` | `longhorn` | storage class enabled in the panel settings on the first start |
| `panel.loadBalancerPool` | `general` | Cilium pool enabled on the first start (name or `lb.cilium.io/pool` value) |
| `panel.helperImage` | `alpine:3.24.2` | image of the file manager pod |
| `panel.timezone` / `panel.dockerInterface` | `UTC` / `127.0.0.1` | passed to game servers / egg config files |
| `panel.trustedProxies` | - | reverse proxy IPs/CIDRs whose `X-Forwarded-For` is trusted |
| `serviceAccount.create` / `rbac.create` | `true` | service account and ClusterRole/Binding (+ Role for secrets) |
| `selfUpgrade.enabled` | `true` | the panel checks `selfUpgrade.chart` every 10 minutes and admins can upgrade from *Settings*; adds the `…-upgrader` service account and roles (see [Self-upgrades](../../docs/installation.md#self-upgrades)) |
| `selfUpgrade.chart` | `oci://ghcr.io/syntax3rror404/charts/kubedactyl` | OCI chart reference without tag |
| `service.type` / `service.port` | `ClusterIP` / `80` | panel service |
| `ingress.*` | disabled | Ingress (`className`, `hosts[].paths[]`, `tls`, `annotations`) |
| `httpRoute.*` | disabled | Gateway API HTTPRoute (`hostnames`, `parentRefs` (default `envoy-gateway-external` in `envoy-gateway-system`), `matches`) |
| `resources`, `nodeSelector`, `tolerations`, `affinity`, `extraEnv` | | as usual |

The release namespace holds eggs, users and the panel settings; users get the namespaces
`<release namespace>-user-<name>`. One release per namespace; several releases in one cluster are
isolated from each other.

Upgrade by hand with `--reset-then-reuse-values` (not `--reuse-values`, which keeps the old chart's
defaults and misses new values), or with the button on the panel's *Settings* page.

`helm uninstall` removes the panel but keeps game servers, user namespaces and volumes. Remove them
with `KUBEDACTYL_NAMESPACE=<namespace> scripts/uninstall.sh` (see [docs/installation.md](../../docs/installation.md#cleanup)).
