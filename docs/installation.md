# Installation and operation

Running the panel in a cluster with the Helm chart, upgrades, configuration and cleanup.

## Running in the cluster

The recommended way is the Helm chart from GHCR (image and chart are public OCI artifacts,
linux/amd64):

```bash
helm install kubedactyl oci://ghcr.io/syntax3rror404/charts/kubedactyl --version 0.2.59 \
  -n kubedactyl --create-namespace \
  --set httpRoute.enabled=true --set 'httpRoute.hostnames[0]=kubedactyl.example.com' \
  --set panel.trustedProxies=10.244.0.0/16
kubectl -n kubedactyl logs deploy/kubedactyl | grep setup   # one-time setup link (use your hostname)
```

Chart values: `charts/kubedactyl/README.md` (Ingress or Gateway API HTTPRoute, admin password /
setup token, initial storage class and pool, …).

**Releasing** (no Docker needed): `make push-image` builds the static linux/amd64 binary and assembles
the image with `scripts/imagebuild` (go-containerregistry: alpine base + `/kubedactyl` + user 65532,
same as the last Dockerfile stage) and pushes `ghcr.io/syntax3rror404/kubedactyl:<appVersion>` and
`:latest`; `make push-chart` writes the version into `Chart.yaml` and pushes the chart. Every build also writes
`third-party-licenses.md` (`make licenses`: the dependencies tracked by hand in the tables of
`THIRD_PARTY_NOTICES.md`; the Vite plugin `frontend/scripts/licenses-plugin.ts` and `backend/tools/licenses` fail
the build when the tables and the bundled npm packages / the Go modules of the linux/amd64 binary differ). It is shown as
Markdown on the public page `/licenses` ("Licenses" in the footer) and stored in the image under
`/usr/share/licenses/kubedactyl/`. Both use `gh auth token`
(scope `write:packages`) without storing it.

**Version:** the file `VERSION` is the only place to change. The build writes it into the binary
(`-X main.appVersion`), from where it reaches `/api/info`, the Swagger UI and the footer of every
page; the image tag, the chart `version`/`appVersion` and the install commands in the README and these docs follow it
(`make sync-version`). Release: edit `VERSION`, `make push-image push-chart`, then
`helm upgrade kubedactyl oci://ghcr.io/syntax3rror404/charts/kubedactyl --version <version> -n kubedactyl --reset-then-reuse-values`
(or the button on the *Settings* page, see below). Use `--reset-then-reuse-values`, not `--reuse-values`: the
latter also keeps the **old chart's defaults**, so values a new chart version adds (e.g. `selfUpgrade`) would
be missing.

### Self-upgrades

With `selfUpgrade.enabled` (chart default) the panel asks the registry in `selfUpgrade.chart` for new chart
versions every 10 minutes, anonymously (the chart is public); without internet the check fails quietly.
Admins see newer versions on the *Settings* page (card *Panel updates*, "Check now") and an "Update" badge
in the sidebar. **Upgrade** starts the Job `<release>-upgrade-<version>-…` in the panel namespace: it runs the
panel image itself (`kubedactyl upgrade --release … --chart … --version …`, Helm SDK built in) and does what
`helm upgrade --reset-then-reuse-values` does: the new chart's defaults plus the values the release was
installed or last upgraded with, only the version changes. Only versions above the running one are offered
(no downgrades, no pre-releases). The Deployment uses `Recreate`, so the panel restarts on the new version
(game servers keep running); the page waits for the new version and reloads. Failed jobs show their last
log lines on the card. Only the last 3 upgrade jobs (with their pods) are kept; older finished ones are removed
after every upgrade and once an hour, all of them 24 hours after they finish.

- Self-upgrades need the admission policy (Kubernetes 1.30+): without it the chart leaves the upgrade account,
  its roles and the button out, and the panel is upgraded with `helm upgrade` only.
- The panel's service account may only create Jobs in its namespace (Role `…-upgrade-jobs`). The job runs as
  `<release>-kubedactyl-upgrader`: everything the chart creates in the release namespace plus the Helm release
  secrets, and `get/update/patch/delete/escalate/bind` on **this release's** ClusterRoles/Bindings and its
  admission policy only (so a new version can change the panel's permissions). That makes the account powerful,
  so only the upgrade job may use it: the admission policy lets the panel create no pod with it and no job but
  exactly `kubedactyl upgrade` with the panel image, the configured chart and a version. Turn
  `selfUpgrade.enabled` off if you only upgrade by hand.
- A chart version that **adds** a cluster role (0.2.31: the tenant role) needs one manual `helm upgrade` with an
  administrator's kubeconfig, because the upgrade account may only change existing roles. The button then fails before
  anything changes (Helm checks the new role first); later versions upgrade by button again.
- The upgrade uses the field manager `helm` like the CLI, so manual and automatic upgrades share ownership
  of the fields (server-side apply). A value `image.tag` pins the image: the upgrade succeeds but the panel keeps
  its version (the card says so).
- Offline / air gapped: `kubedactyl upgrade --chart-file chart.tgz` upgrades from a packaged chart, or point
  `selfUpgrade.chart` to a registry mirror.
- Verified on the cluster with a test release: the job's service account (impersonated and in a pod with the
  job's security context) upgraded the release, kept the user values, applied a new chart default and added a
  permission to the panel's ClusterRole; the version check lists the GHCR tags anonymously.

Without a kubeconfig the panel uses the **mounted service account** (in-cluster config; the cluster
page then shows mode `in-cluster`, the service account and its token expiry). The chart creates it
with the permissions the panel needs, split so a compromised panel cannot reach beyond its own namespaces:

- **ClusterRole `<release>`** (cluster wide): read what it caches (pods, services, claims, config maps, network
  policies), nodes, metrics, storage classes, pools; write only what has no namespace (its user namespaces,
  their volumes for reclaim policy and transfers, its own CRDs) plus role bindings to its tenant role.
- **ClusterRole `<release>-tenant`**, bound by the panel in every user namespace: pods (with exec/attach/logs),
  services, volume claims, config maps, network policies. Nowhere else.
- **Role `<release>-panel`** in the release namespace: its session key secret, short-lived hardware probe
  pods (logs, no exec) and the config map `kubedactyl-node-hardware` with their results.
- **ValidatingAdmissionPolicy `<release>-panel`** (Kubernetes 1.30+): the API server rejects requests of the
  panel's service account outside its user namespaces (namespaces, role bindings, volumes), foreign CRDs, pods
  with a service account token, host namespaces, host paths or privileged containers, and every job except the
  upgrade job. The sidebar health card warns when it is missing.

The Deployment runs a single replica (non-root, read-only root filesystem, probes on
`/api/health`). With Docker the image can also be built from the `Dockerfile`
(Node → Go → Alpine 3.24.2 as non-root user 65532, the frontend is embedded):

```bash
docker build --build-arg VERSION=$(cat VERSION) -t ghcr.io/syntax3rror404/kubedactyl:$(cat VERSION) .
```

Verified on the cluster with a test release and the panel running as its service account (impersonated):
all 20 permission checks, user namespaces with labels, tenant binding and network policy, a real egg
installation, the files helper as UID 988, files, archives, backups and restores. As the same account the API
server rejected pods in `kube-system`, role bindings outside user namespaces or to `cluster-admin`, foreign
namespaces, volumes and CRDs, pods with a service account token or host access, and every job except the
upgrade job generated by the code. The Dockerfile itself was not built here (no container runtime on the
development machine).

## Configuration

Flags or environment variables:

| Flag | Env | Default | Meaning |
|------|-----|---------|---------|
| `--kube-context` | `KUBE_CONTEXT` | current context / in-cluster | kubeconfig context (`make` passes `KUBE_CONTEXT`, e.g. from `local.mk`) |
| `--namespace` | `KUBEDACTYL_NAMESPACE` | `kubedactyl` | panel namespace (eggs, users, settings); game servers run in `<namespace>-user-<user>` |
| `--storage-class` | `STORAGE_CLASS` | `longhorn` | StorageClass enabled in the panel settings on the first start |
| `--lb-pool` | `LB_POOL` | `general` | pool enabled on the first start (pool name or its `lb.cilium.io/pool` label value) |
| `--helper-image` | `HELPER_IMAGE` | `alpine:3.24.2` | image of the file helper and hardware probe pods |
| `--docker-interface` | `DOCKER_INTERFACE` | `127.0.0.1` | value of `{{config.docker.interface}}` in egg config files |
| `--timezone` | `TZ` | `UTC` | `TZ` passed to game servers |
| `--listen` | `PORT` | `:8080` | HTTP listen address |
| - | `KUBEDACTYL_ADMIN_PASSWORD` | - | creates `admin` with this password instead of the setup page (only while no admin exists) |
| - | `KUBEDACTYL_SETUP_TOKEN` | generated, logged | token for the setup page |
| `--trusted-proxies` | `TRUSTED_PROXIES` | none | comma separated IPs/CIDRs of reverse proxies whose `X-Forwarded-For` is used as client IP (login throttling) |
| `--verbose` | `VERBOSE` | off | debug logging |
| - | `KUBE_HTTP2` | off | use HTTP/2 for the Kubernetes API (see below) |

The Kubernetes client uses **HTTP/1.1** by default: proxies in front of the API server often cut
long-running HTTP/2 streams after a few seconds with a stream error, which client-go answers with
backoff (missed events). Over HTTP/1.1 the same cut is a clean end of the watch and it reconnects at once.

## Cleanup

`make uninstall` removes the panel namespace (eggs, users, auth secret) and every user namespace of
the installation with all game servers, pods, services and volumes (the data is deleted), even when
the panel is not running:

```bash
make uninstall                                       # context from the Makefile, asks for confirmation
YES=1 DELETE_CRDS=1 make uninstall                   # no prompt, also delete the CRDs (all namespaces!)
KUBEDACTYL_NAMESPACE=kubedactyl-test make uninstall  # another namespace
```

It switches the PVs to reclaim policy `Delete` (Longhorn `longhorn` retains volumes otherwise),
releases the `GameServer` and `User` finalizers, deletes the namespaces and waits until the volumes
are gone. Only namespaces named `<namespace>-user-*` with the panel label are touched, so other
installations in the same cluster stay untouched.

## Known limitations

- Egg import from a URL (admins only) fetches arbitrary URLs from the panel.
- The login throttle is kept in memory (reset on restart); the panel sends no e-mails (invites are links or QR
  codes the administrator passes on) and has no self-service password reset.
- Egg install scripts and many images need internet access from the cluster.
- `{{config.docker.interface}}` has no real equivalent in Kubernetes; it resolves to `--docker-interface`.
- Backups are stored on the server volume only (no off-site copy); download important ones.
- Self-upgrades only move forward and only with chart versions that keep the chart name; a new chart version
  that adds a ClusterRole under another name needs one manual `helm upgrade`.
- Job history (backups, restores, downloads) is kept in memory; a panel restart interrupts running jobs.
- Console history and file activity are kept in memory: after a panel restart the console starts
  empty for stopped servers.
- File activity: after a panel restart running files pods are removed at the next
  check and start again on the next file operation.
- Deleting a server deletes its data: the PV reclaim policy is switched to `Delete` before the PVC goes away.
- Swagger docs use Swagger 2.0 (swag v1); swag v2 (OpenAPI 3.1) is still a release candidate.
