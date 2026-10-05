# Features

Everything the panel does, in detail. The short overview is in the [README](../README.md).

## Overview

| Area | What works |
|------|------------|
| Eggs | Import `PTDL_v1`, `PTDL_v2`, `PLCN_v1` to `v3` as JSON or YAML (file upload or URL). All 502 eggs of `pelican-eggs/{minecraft,games-steamcmd,games-standalone}` parse. Create eggs in a step-by-step wizard, edit them (configuration, images & startup, process management / config files, variables, install script), duplicate, "Copy settings from" another egg, update from the egg's update URL, export (see *Eggs*) |
| Install | Egg install script runs as a pod (installer image, script at `/mnt/install/install.sh`, data at `/mnt/server`), output streamed to the console |
| Start | The egg's config file changes (parsers `file`, `yaml`, `properties`, `ini`, `json`, `xml`), ownership fix, then the game pod (fixed user, read-only root file system) |
| Console | Live output over websocket (xterm.js), commands via TTY stdin, "done" detection → *Running*, history survives panel restarts |
| Power | Start, stop (egg stop command or `^C`/`^^C`/signal, killed after the stop timeout), restart, kill; crash detection with auto restart (not twice within 60 s) |
| Files | Browse, edit (CodeMirror), upload, download, rename/move (also drag and drop onto folders and path segments; never overwrites), delete, compress (tar.gz), extract (zip/tar), download from a URL (runs in the file container); the file container only runs while it is used; a light on the Files tab shows it (green running, red stopped, green/yellow starting) and the file page counts down until it stops without file operations |
| Backups | tar.gz of all server files in the `.backups` folder of the volume, optional label, download, restore (deletes everything except `.backups`, then unpacks), delete; also as a schedule task |
| Transfer (admin) | Moves a stopped server with its files and backups to another user; name and address stay |
| Branding (admin) | Name, tagline and logo in the sidebar and on the sign-in page, browser title and favicon (*Settings → Branding*; images as PNG, JPEG, GIF, WebP, SVG or ICO up to 128 KiB). Without a favicon the browser shows its default; the footer always names the software |
| Suspend (admin) | Stops a server and locks it for its owner (visible, but no console, files, backups, settings or power); admins keep access |
| Network | Every port as TCP **and** UDP on a `LoadBalancer` service from a selectable Cilium LB IPAM pool, optional fixed IP (checked against the pool), external traffic policy per server (admins: `Local`, the default, where the server sees the players' IPs, or `Cluster`); users see an external domain instead of the IP |
| Tasks | Console commands at events instead of times (a tab next to *Schedules*): after the server was marked as running, or before it stops or restarts. The server stays online until the commands and their delays (up to 5 minutes) are done. Not on kill or a crash |
| Diagnostics | A tab per server that checks the usual reasons why players cannot connect: installation, state (crash, out of memory, suspended), pending restart, disk space, server address, external domain, DNS A/AAAA records as the internet sees them (asked at the domain's authoritative name servers, so a local DNS override does not matter), and the router forwarding per port (for games that accept TCP: through the public IP, tested from the server's network, i.e. NAT loopback; UDP-only games get a note, UDP cannot be tested). Users do not see internal addresses |
| Storage | Selectable StorageClass per server (fixed for the life of the volume) |
| Stats | CPU / memory from metrics-server, disk usage of the volume, uptime |
| Users | Admins and users, one namespace per user, Argon2id passwords, sessions and API tokens (see [Administration](administration.md#users-and-permissions)) |
| Settings (admin) | External domain, storage classes and load balancer pools users can pick (all of the cluster listed, just check them) |
| Panel updates (admin) | The panel looks for new chart versions in its OCI registry every 10 minutes; *Settings* shows them (plus an "Update" badge in the sidebar) and upgrades with one click: a Job runs `helm upgrade` with the release's values and only the new version |
| Cluster (admin) | Nodes with status, roles, CPU model / cores / threads / clock, vendor and product, memory, live usage and requested resources, totals; kubeconfig or service account in use, authentication method, identity (SelfSubjectReview) and the permissions the panel needs |
| Egg features | All keys Pterodactyl/Pelican know: `eula` (accept the EULA), `java_version` (switch to another image of the egg), `pid_limit` and `steam_disk_space` (explain the error, admin/user texts), `gsl_token` (enter a new `STEAM_ACC`), `hytale_oauth` (login link). Console output is matched with the same patterns, dialogs only while the server is not running and only for the latest run |
| Schedules | Cron expression (or `@daily` …) in the panel time zone, console commands, power actions and backups with delays, "only when online", run now, last result; managed by the server owner |
| Setup & security | Setup page with one-time token on the first start, CSP and security headers, CSRF check, network isolation of user namespaces (see [Security](security.md)) |
| Comfort | "Restart required" banner when runtime settings changed while running, admin notice shown every time a user opens a server, "Sign out everywhere", tab titles per page, dashboard search/status filter (more than 6 servers), drag and drop to move files and to upload from the desktop, unsaved-changes guard and ⌘/Ctrl+S in the editor, console history (↑/↓) |

## Eggs

Admins manage eggs under *Eggs*, nobody has to write egg files by hand:

- **Library** (tab next to *Installed*, with the same search field): the eggs of the GitHub repositories set under
  *Settings → Egg library* (e.g. `https://github.com/pterodactyl/game-eggs`), shown like the installed ones. A click opens
  the egg (overview, variables, config files, install script) with **Install** and **Update automatically**; installing
  imports the raw file and keeps its URL as update URL. The panel downloads each repository as one archive
  (`codeload.github.com`, no GitHub API rate limit), keeps the egg summaries in memory for 15 minutes (*Refresh*
  downloads again) and stores nothing of the library in the cluster. Pelican repositories carry every egg twice
  (`egg-x.yaml` and `pterodactyl-egg-x.json`); the library lists the Pelican file.

- **New egg** is a wizard (configuration → images & startup → process management → variables → install script →
  review); **Edit** shows the same parts as tabs, **Duplicate** starts the wizard with a copy (" Copy", no update URL,
  new UUID, like Pelican's "Replicate"). "Copy settings from…" takes stop command, startup detection and config files,
  or the install script, from another egg once (no inheritance as in Pterodactyl/Pelican).
- The API validates like the two panels, so exports stay importable there: author is an e-mail address, environment
  variables match `^\w{1,191}$` and are not reserved (Pterodactyl's and Pelican's lists), rule strings are checked
  (regular expressions, missing arguments), config files are paths inside the server folder. Errors appear at the field
  and as counters on the tabs/steps.
- Changes apply to servers from their next start (install script: next reinstall); servers keep their own image,
  startup override and variable values.
- **Export** shows the file with a switch **YAML | JSON | Pterodactyl**: YAML and JSON are the same Pelican egg
  (`PLCN_v3`, key order of Pelican's exporter, `config.*` as maps in YAML and as JSON strings in JSON); *Pterodactyl* is
  `PTDL_v2` for importing into Pterodactyl (no UUID, tags or icon; rules as `a|b` string). Every egg keeps a UUID
  (imported, generated, or derived from its name for older eggs), so Pelican updates the same egg on a re-import.
- **Update automatically** (a switch next to the update URL): the panel checks the URL every hour and applies a newer
  file and replaces changes made in the panel; the egg shows "Auto update", when it was checked and updated, or the error.
- **Update from URL** (eggs with a URL show an "Update URL" badge) downloads `meta.update_url` again and replaces all fields (changes made in the panel are lost, as
  the dialog says so); name and UUID stay.
- **Update URL:** `meta.update_url` of the file, or the URL it was imported from when the file has none (many egg
  files have none). It can be changed or cleared in the editor; clearing it removes the link completely (also the
  import URL and the auto update status), and the egg is no longer updated.
- **Restart required:** editing startup, variable defaults or config files of an egg flags running servers
  of that egg.
- Not supported: "Force Outgoing IP" and log configuration (no meaning on Kubernetes), several named startup commands
  (the first one is used).

API: `POST /api/eggs`, `PUT /api/eggs/{egg}`, `GET /api/eggs/{egg}/export?format=yaml|json|ptdl[&download=true]`,
`POST /api/eggs/{egg}/update-from-url` (admins).

## How a server runs on Kubernetes

| Part | Kubernetes object |
|------|-------------------|
| Egg | `Egg` custom resource |
| Server | `GameServer` custom resource |
| Server data | PVC `<server>-data` (StorageClass selected per server) |
| Address and ports | `LoadBalancer` service `<server>` with TCP+UDP per port, labeled with the service selector of the selected pool |
| Panel settings | `PanelSettings` custom resource `panel` in the panel namespace |
| Installation | Pod `<server>-install` (runs as root; its exit code is recorded but does not fail the installation) |
| Game process | Pod `<server>-game`: UID/GID 988, TTY + stdin, read-only root FS, 100 MiB tmpfs `/tmp`, memory limit + runtime overhead |
| File access | Pod `<server>-files` (tiny alpine helper, started on demand, removed after 1 idle minute) |
| Console | Kubernetes attach (stdin) and log streaming (output) |

Why a helper pod: the file manager and the pre-start steps (config parsers, `chown`) must work while
the game server is stopped. It only runs while needed, so many idle servers do not keep pods running:

- Opening the file manager calls `POST /files/session`; the UI shows a waiting view ("your file
  container is spinning up") and keeps asking with `POST /files/session` until the container was ready once,
then polls `GET /files/session` (reading the state is no activity). Every file
  operation counts as activity and waits up to 45 s for the container (so a save after a long edit
  still works). Starting a server also starts it for the pre-start steps.
- A reaper (checks every 10 s) removes files pods after **1 minute without file activity**; the UI then shows "stopped"
  with *Start again*. Before removal the disk usage is measured and stored in
  `status.diskUsedBytes` (the dashboard shows it while neither the file container nor the game runs;
  a running game is measured inside the game container).
- Longhorn RWO volumes can be mounted by several pods, but only on one node. Game, install and files
  pods carry `kubedactyl.io/volume=<server>` and a required pod affinity on that label. A pod matches
  its own term, so the first one may start on any node and the others join it; terminated pods are
  ignored by the scheduler (both verified live).
- The files pod is read uncached: the informer can lag by minutes behind API proxies that cut watches.
  For the same reason the controller decides "create or update" for volumes, services, install pods
  and config maps, network policies and the panel settings with an uncached read
  (`kube.CreateOrPatch`), and the install pod only starts once the volume is bound.

## Schedules

Stored in the GameServer (`spec.schedules`, results in `status.schedules`), so they are deleted with
the server. A runner in the panel (`internal/schedule`) checks all schedules at the start of every
minute in the panel time zone (`--timezone` / Helm `panel.timezone`; the time zone database is
embedded) and runs the tasks of due schedules one after another: console command,
start/stop/restart/kill or a backup (payload = optional label), each after its delay (max. 900 s).
Suspended servers are skipped. A run that is still in progress is not
started twice; runs missed while the panel was down are skipped. Limits: 20 schedules per server,
10 tasks per schedule. API: `GET/PUT /api/servers/{name}/schedules`, `POST …/schedules/{schedule}/run`.

## Backups, downloads from a URL, suspend

**Backups** live on the server volume in the folder `.backups` (next to the server files, counted towards
the disk size, deleted with the server). Creating one runs `tar -czf` in the file container over
everything except `.backups` into a temporary file that is renamed when complete. Restoring requires a
stopped server: it deletes everything in the volume except `.backups` and unpacks the archive (files
owned by 988:988 afterwards). Starting the server is refused while a restore runs; backup and restore
exclude each other per server. Names: `backup-<date>_<time>[-label].tar.gz` in the panel time zone.

**Download from a URL** (file manager → *From URL*) runs `wget` in the file container, not in the panel:
the network policy of the user namespace applies (no private networks unless the admin allows them), so the
panel cannot be used to reach internal services. Only http/https; existing files are not overwritten; the
file name comes from the URL or is given.

Backups, restores and downloads run as **jobs** in the background (`GET /api/servers/{name}/jobs`, last 20
per server, kept in memory, max. 1 hour); they keep the file container alive while they run, and the
page announces finished jobs.

**Suspend** (`POST /api/servers/{name}/suspend`, admins) sets `spec.suspended`: the server is stopped and
cannot be started (also not by schedules); its owner still sees it but every other server endpoint
answers 403. Admins keep full access except starting.

API: `GET/POST /api/servers/{name}/backups`, `POST …/backups/{backup}/restore`,
`DELETE …/backups/{backup}`, `POST …/files/pull`.

## Resources and scheduling

Every pod has requests and limits so the Kubernetes scheduler can place it sensibly: **memory
request = limit** (memory cannot be taken back), **CPU request = half the limit** (bursts up to the
limit stay possible). Game pod: the server's CPU/memory (memory plus a runtime overhead); a server
with unlimited CPU reserves 1 core without a limit. Install pod: 2 cores / at least 1 GiB, files
pod: 0.5 cores / 128 MiB. The only placement rule is that all pods of one server share a node (RWO
volume); everything else is decided by the Kubernetes scheduler.

The game pod carries a hash of its runtime settings (image, startup, variables, memory, CPU,
ports); when the spec differs while it runs, `status.restartRequired` is set and the server page shows
a *Restart required* banner with a restart button.

Finished pods do not pile up: a game pod is deleted 3 seconds after its process ended (the crash
detection has run and open consoles got the last lines), a finished install pod and its script
ConfigMap once the installation is recorded. The console history is kept in memory while the panel
runs.
