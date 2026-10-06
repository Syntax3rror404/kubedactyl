# Administration

Users, permissions and the settings administrators manage in the panel.

## Users and permissions

Users are `User` resources in the panel namespace; every user owns the namespace
`<panel namespace>-user-<username>` (default `kubedactyl-user-<username>`) that holds its game servers.
Passwords are stored as **Argon2id** hashes (PHC format), never in clear text.

**First start:** while no active administrator exists, every page redirects to **`/setup`**, where the
first administrator chooses username and password and is signed in. The setup needs a one-time
**setup token**: the panel logs a link on start (`open the setup page with this one-time link
url=…/setup?token=…`) or uses `KUBEDACTYL_SETUP_TOKEN`, so nobody who merely reaches the page can take
over the panel. For automated installations `KUBEDACTYL_ADMIN_PASSWORD` creates `admin` directly
instead. After the setup the page answers 409.

**New users:** administrators add users on the *Users* page in two ways:
- **New user** with a start password. *Must choose a new password after signing in* (on by default,
  `mustChangePassword`) sends the user to `/change-password` after the sign-in; until the password is replaced
  every request except the own account, the password change and signing out answers 403, for sessions and API
  tokens alike. Administrators can set the switch again for any user.
- **Invite**: a one-time link (`/invite?token=…`), shown once as QR code and as text to send in a messenger. The
  invited person chooses username (unless the administrator fixed one), display name and password and is signed in.
  A link works once, for 7 days; open invites are listed below the users, where *Renew* replaces the link with a
  new one valid for 7 days again (also after it expired; the old link stops working) and the trash can revokes it. Invites are `Invite`
  resources in the panel namespace; only a SHA-256 hash of the link is stored. Failed link checks count against
  the client like failed sign-ins.

**Authentication**
- Web UI: `POST /api/auth/login` sets the HttpOnly, SameSite=Strict cookie `kd_session`
  (HMAC-signed). The signing key lives in the secret `kubedactyl-auth`.
- API clients: `Authorization: Bearer <token>` with the session token from the login response or a
  personal **API token** (`kdt_…`, created under *Account*, only a SHA-256 hash is stored, shown once).
- Lifetimes (*Settings → Sessions and API tokens*): a sign-in lasts 12 hours, an API token at most 90 days
  (1-720 hours / 1-3650 days). Both are checked on every request, so a shorter value also ends older sessions
  and tokens (counted from their creation); tokens without an expiry no longer exist.
- Changing or resetting a password and disabling a user invalidates all sessions of that user;
  *Account → Sign out everywhere* (`POST /api/auth/logout-all`) does the same on request (API tokens
  stay valid). A password change also revokes the own API tokens unless *Also revoke my API tokens* is switched
  off (`revokeTokens`); an administrator's password reset always revokes them. API tokens cannot create tokens
  (403); only a session can.
- Failed logins: 5 per client IP and username within 5 minutes, 20 per client IP and 30 per username within
  15 minutes → HTTP 429. Attempts count before the password is checked, so parallel requests get no more tries.
  A browser in which the user signed in before (cookie `kd_device`, 180 days, sent only to the sign-in) is exempt
  from the per-username limit: failed attempts from elsewhere cannot lock its user out.

| Area | User | Admin |
|------|------|-------|
| Own servers: view, power, console, commands, files, backups, stats, reinstall | ✓ (not while suspended) | ✓ (all servers) |
| Suspend / unsuspend servers | - | ✓ |
| Transfer servers to another user | - | ✓ |
| Server settings: display name, crash restart, egg image, **editable** egg variables, load balancer pool (enabled pools) | ✓ | ✓ |
| Server resources, ports, IP, startup command, stop timeout, custom image | - | ✓ |
| Create / delete servers (with owner) | - | ✓ |
| Eggs: list / view (hidden variables and install script stripped) | ✓ | ✓ full |
| Eggs: import / delete, egg library | - | ✓ |
| Users: create, invite, edit, disable, reset password, delete (with servers, data, namespace) | - | ✓ |
| Cluster page: nodes, hardware, capacity, connection and permissions | - | ✓ |
| Panel settings: external domain, enabled storage classes and pools (read: everyone) | - | ✓ |
| Own account: password, API tokens | ✓ | ✓ |

Servers of other users answer **404** (their existence is not revealed). Variables an egg marks as
not user-viewable are removed from responses for users. The last active administrator cannot be
deleted, demoted or disabled, and nobody can delete the own account.

Several installations can share a cluster (different `--namespace`): each one only reconciles and
shows objects in its own namespace and its own `<namespace>-user-*` namespaces.

## Panel settings

*Settings* (admins) is stored in the `PanelSettings` resource `panel` in the panel namespace
(`kubectl -n kubedactyl get kdsettings`).

- **Egg library:** GitHub repositories (`https://github.com/<owner>/<repo>`, at most 20) whose eggs the
  *Library* tab of *Eggs* lists.
- **Kube API limit:** requests per second the panel sends to the Kubernetes API at most (5-1000, default 50,
  bursts of twice that) and requests per second one user may send to the panel (*Per user*, 1-200, default 10).
  Protects the API server from overload; both apply at once, without a restart. All clients of the panel share one
  limiter (`kube.RateLimiter`). A request is refused before it starts, never in the middle: 429 when its user sent too
  many, 503 ("The panel is busy") while the calls already waiting for the limit would take longer than 3 s
  (administrators are not refused as busy but wait in the queue, so they can always raise the limit). So no
  long queue builds up and no change is left half done. Both answers carry `Retry-After`; the web interface shows a
  notice and keeps polling. The footer shows the user's requests per second against the per-user limit (and, for
  administrators, the panel's Kubernetes API calls against the panel limit) with a green, yellow (from 50 %) or red
  (from 90 %) light: averages over 5 s from `GET /api/request-rates`, which does not count itself. Measured with the default: 50 users with an open server page answer in ~55 ms (a server
  page costs about one API request per second); one user flooding the panel gets ~10 requests per second through and
  other users do not notice it.
- **Legal notice & privacy policy:** Markdown (max. 20,000 characters each, live preview), linked
  in the footer of every page including sign-in and setup; served publicly by `GET /api/legal` (only
  these two texts). Rendered with react-markdown: no raw HTML, unsafe links are dropped.
- **Server notice:** plain text (max. 2000 characters) that users see as a dialog every time they
  open one of their servers (switching tabs of the same server does not show it again; admins do not
  get it).
- **External domain:** users see `domain:port` instead of the load balancer IP; the API replaces
  `status.address` with the domain (and drops a fixed IP) in responses for users. Admins still see the
  IP. The domain must resolve or be forwarded to the IPs of the enabled pools.
- **Storage classes:** every StorageClass of the cluster is listed (provisioner, reclaim policy,
  binding, expansion); the checked ones appear in the storage dropdown of *New server*, one is the
  default. `spec.storageClass` of a server cannot change later (CEL rule on the CRD).
- **Load balancer pools:** every `CiliumLoadBalancerIPPool` (`cilium.io/v2`, falls back to `v2alpha1`)
  with blocks, free IPs and its service selector. A server stores the pool **name**; the controller
  sets the `matchLabels` of the pool's `serviceSelector` on the service (e.g. pool `general-pool` →
  `lb.cilium.io/pool=general`) and removes the labels of the previous pool when it moves
  (annotation `kubedactyl.io/pool-labels`). Pools that are disabled, use `matchExpressions` or select
  by service namespace/name cannot be enabled. Users can move their servers between enabled pools
  (the address changes; Cilium releases the old IP and assigns a new one, verified live). A fixed IP
  must be inside the pool's blocks and is cleared when the pool changes.

- **Panel updates:** see [Self-upgrades](installation.md#self-upgrades).

On the first start the settings are created from `--storage-class` and `--lb-pool` (pool name or the
value of its `lb.cilium.io/pool` selector label), and servers without a storage class / pool are
filled in from their existing PVC and service.

## Cluster page

*Cluster* (admins) shows every node with roles, status, pressure conditions, taints, OS / kernel /
kubelet / runtime, capacity, **live usage** (metrics-server) and the **resources requested** by running
pods, plus the totals of the cluster.

Kubernetes does not know the CPU model or the machine. The panel reads it with a short-lived
**probe pod per ready node** (`kubedactyl-probe-<node>-…` in the panel namespace): unprivileged
(UID 65534, read-only root filesystem, all capabilities dropped, tolerates every taint), it prints
`/proc/cpuinfo` (model, cores, threads, max clock, hypervisor flag) and `/sys/class/dmi/id`
(vendor, product, BIOS) and is deleted right away. Results are cached for 24 h in the ConfigMap
`kubedactyl-node-hardware`; *Re-read hardware* probes again (`POST /api/cluster/nodes/probe`).
Nodes that are not ready are skipped.

**Health checks** (`GET /api/cluster/health`, cached 30 s, checked again at once after a settings change) are
shown in the sidebar cluster card above the user menu (green / amber / red; problems are always listed with
their explanation, hovering the card also lists the passing checks in green): Kubedactyl CRDs served, all
permissions granted, nodes ready, metrics-server available, Cilium LB IPAM installed and the enabled pools
present with free addresses, the enabled storage classes present, and (in the cluster) the admission policy
that limits the panel's permissions ("Permission guard").

The *Access* section shows how the panel connects: kubeconfig file, context, cluster, user entry and
all context names (or the in-cluster service account and its token expiry), the authentication
method (client certificate subject/expiry, masked token, exec plugin with credential arguments
redacted), TLS/CA, the identity the API server reports (`SelfSubjectReview`) and a
`SelfSubjectAccessReview` for every permission the panel needs.
