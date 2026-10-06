# Development

Working on the code. Read [ARCHITECTURE.md](../ARCHITECTURE.md) first for the big picture.

## Local setup

Requirements: Go ≥ 1.27, Node ≥ 22, a kubeconfig with cluster-admin rights (CRDs, namespace, PV patch).

```bash
cd frontend && npm ci && cd ..
make dev                            # UI http://localhost:5173 (hot reload), API :8080
make run                            # production build → http://localhost:8080
make run KUBE_CONTEXT=my-cluster    # another kubeconfig context
```

On start the panel applies its CRDs (server-side apply), creates the namespace and, on the very
first start, the user `admin` (password in the log, see [Users and permissions](administration.md#users-and-permissions)).

1. Sign in at http://localhost:8080/login as `admin`
2. **Eggs → Import egg**, e.g. `https://raw.githubusercontent.com/pelican-eggs/minecraft/refs/heads/main/java/paper/egg-paper.yaml`
3. **Users → New user** (optional), then **New server** → pick the egg and the owner, size it, *Create server*
4. Watch the installation in the console; Minecraft asks for the EULA, accept it in the dialog

## Commands

Commands that talk to a cluster (`make dev`, `make run`, `make uninstall`) need an explicit kubeconfig context,
so they never use the current one by accident. Put it into `local.mk` (not versioned):

```make
KUBE_CONTEXT := my-cluster
```

| Command | Description |
|---------|-------------|
| `make dev` | Vite (5173) + Go API (8080) with `-tags dev`; Vite proxies `/api` (incl. websocket) and `/swagger` |
| `make build` | generate CRDs + deepcopy, Swagger docs, build frontend, build `bin/kubedactyl` |
| `make run` | build and start |
| `make test` | Go unit tests (works in a fresh clone, before the first frontend build) |
| `make lint` | code rules: golangci-lint (`backend/.golangci.yml`), unused Go code, frontend checks (see below) |
| `make fmt` | rewrite the formatting: gofmt + goimports, Prettier |
| `make generate` | `controller-gen` for `backend/api/v1alpha1` → `backend/config/crd` |
| `make docs` | Swagger docs from the handler comments → `backend/docs`, and the frontend API types generated from them → `frontend/src/lib/types/api.gen.ts` |
| `make check-generated` | Regenerates CRDs, deepcopy, Swagger and API types and fails if they were out of date (part of `make test`) |
| `make ui-update` | re-install all library components from their registries (see below) |
| `make uninstall` | delete everything the panel created in the cluster (see [Cleanup](installation.md#cleanup)) |
| `cd frontend && npm run e2e` | browser smoke test against the running panel (all pages, both themes) |
| `cd frontend && npm run format` | format the frontend with Prettier (`npm run format:check` only checks) |

The e2e test signs in through the login page: `E2E_USER=… E2E_PASSWORD=… npm run e2e`.
`E2E_SERVER=<name>` additionally starts the server, sends a command, creates and deletes a
folder and stops it again. Cluster integration test: `KUBE_IT_CONTEXT=<context> go test ./internal/kube/ -run Attach -v`
(creates and deletes the namespace `kubedactyl-attachtest`).

Code hygiene (no warnings expected): `make lint` runs
[golangci-lint](https://golangci-lint.run) with `backend/.golangci.yml` (go vet, staticcheck, errcheck, unused code,
lines of at most 120 characters, functions of at most 60 lines / 40 statements (tests excepted), no cognitive
complexity above 20 (except the ported config file parsers in `internal/configfile`), gofmt and goimports), `deadcode`
(unreachable functions; `internal/testutil` is only used by tests) and the frontend checks
`npm run format:check && npx tsc -b && npx oxlint && npx knip` (unused files, exports and dependencies; library files are
ignored via `knip.json`). `make fmt` rewrites the formatting (Go: gofmt + goimports, frontend: Prettier); long lines are
wrapped by hand.

## Library components and updates

Files that come from a library are **never edited by hand**, so they can be updated at any time:

| Files | Source | Update |
|-------|--------|--------|
| `frontend/src/components/ui/*` (except the two below) | shadcn registry | `make ui-update` |
| `frontend/src/components/ui/magic-card.tsx`, `light-rays.tsx`, `particles.tsx` | [Magic UI](https://magicui.design) registry (`@magicui/…`, MIT) | `make ui-update` |
| `frontend/src/components/theme-provider.tsx`, `mode-toggle.tsx` | code blocks of the [shadcn Vite dark mode docs](https://ui.shadcn.com/docs/dark-mode/vite) | `make ui-update` |
| `frontend/src/hooks/use-mobile.ts`, `frontend/src/lib/utils.ts` | installed by shadcn together with `sidebar` / `init` | with the components |

The list lives in `frontend/ui-components.json`. `make ui-update` runs `frontend/scripts/update-ui.mjs`:
it re-adds every entry with `--overwrite` and fetches the two docs files. The files stay exactly as the
sources deliver them (some keep a `"use client"` directive, which Vite ignores), and their dependencies get the
versions the registry items name (`chart` pins `recharts`; a newer pin arrives with the next update). The CLI lists
the files it updated (identical ones are skipped). Unused code inside
library files is not touched: TypeScript only
checks types, unused code is reported by oxlint (`npm run lint`, part of `npm run build`), which
ignores the library files.

Running the update twice produces no diff. Earlier hand edits were replaced by wrappers:

| Former edit | Now |
|-------------|-----|
| `ui/sonner.tsx` read the theme from `next-themes` → changed to our provider | original file; `app/app-toaster.tsx` passes the theme of our `ThemeProvider` (the registry Toaster spreads props after `theme`), `next-themes` stays installed as its dependency |
| unused imports in registry files | original files; unused-code checks moved from `tsc` to oxlint, which skips library files |
| `mode-toggle.tsx` labels translated to German | original English labels |
| `sidebar-07` block demo files (`team-switcher`, `nav-*`) | replaced by our own `components/layout/app-sidebar.tsx`; blocks are starting points, not updated |

When adding a registry component: install it with the CLI (`npx shadcn@latest add <name>` or
`@magicui/<name>`) **and** add it to `ui-components.json`. Only these two registries are used (both MIT);
app specific looks go into wrappers, e.g. `components/common/glow-card.tsx` (Magic UI card in the brand colors)
and `components/common/file-dropzone.tsx` (own drop zone on `react-dropzone`).

## Project layout

Rule of thumb: **one folder per topic**. In the backend every package under `internal/` has one job;
file name prefixes group the files of a package (`servers_*.go`, `files_*.go`, `gameserver_*.go`).
In the frontend every feature has its own folder with its pages (`*-page.tsx`) and the components,
hooks and helpers only it uses; shared pieces live in `components/` and `lib/`.

```
backend/
  main.go · wiring.go     start-up: flags, logging, prepare the cluster · build controllers, services and API, HTTP server
  logging.go              quiet slog handler for client-go noise
  api/v1alpha1/           CRD types: Egg, GameServer, User, PanelSettings (kubebuilder markers)
  config/crd/             generated CRD manifests (make generate), embedded and applied on start
  docs/                   generated Swagger (make docs)
  web/                    embedded frontend build
  internal/
    bootstrap/            start-up: CRDs, panel namespace, session key, first admin / setup token, cache config
    httpserver/           router (API, Swagger UI, SPA), security headers/CSP, body limit, trusted proxies, health/info
    httpapi/              REST + websocket handlers, grouped by prefix:
                            api.go (routes) · errors.go (a.fail, status mapping) · auth*.go (login, account, setup) · users.go
                            servers*.go (list/get/delete, create, update, access rules, power/reinstall/suspend, console websocket, stats, backups)
                            files_*.go (read, write, session) · eggs.go · settings.go · cluster.go · upgrade.go
    controller/           reconcilers: gameserver*.go (entry, children, install, game, cleanup of finished pods)
                            · files_reaper.go (idle files pods) · user*.go (user namespaces, network policy)
    gameserver/           Kubernetes objects of a server (pods, PVC, service, placement, resources), validation, pre-start steps
    serverctl/            power actions, console commands, reinstall, suspend, transfer (shared by API and schedules)
    schedule/             cron schedules of servers
    eggstore/ · users/    storing eggs (create, import, update from URL, delete) · accounts (create, change, last-admin rule)
    validation/           field validation errors (422 with messages per field)
    console/              output hub (history, subscribers), log following, "done" detection
    files/                file manager via exec in the files pod, activity tracker, background jobs, backups, URL downloads
    settings/             panel settings, storage classes, Cilium pools
    clusterinfo/          cluster page: nodes, hardware probes, identity, permission checks
    selfupgrade/          panel updates: registry check, upgrade job, `kubedactyl upgrade` (Helm SDK)
    egg/ · configfile/    egg parsing, validation, export (PLCN_v3 / PTDL_v2) · config file parsers of eggs
    auth/ · tenancy/      passwords, sessions, API tokens · user namespaces, installation scoping
    kube/                 exec / attach helpers
frontend/
  ui-components.json      library component manifest (make ui-update)
  scripts/ · e2e/         UI update scripts · browser smoke test (npm run e2e)
  public/theme-init.js    theme before first paint (external because of the CSP)
  src/
    main.tsx              entry: providers
    app/                  router (URL → page), toaster, 404 page
    features/
      auth/               login-page, setup-page · require-auth, auth-shell
      dashboard/          dashboard-page · server-card, server-filter
      servers/            server-layout, server-new-page and one page per server tab (URL /servers/:server/<tab> →
                          <tab>-page.tsx): console, files, startup, schedules, backups, settings
                            · components (server form, power, console, stats, resources, ports, placement, file-* for
                              the file manager, …) · hooks · egg-features (eula, java_version, …) · lib
      eggs/               list, detail, edit (tabs) and new (wizard) pages; components/editor = shared form sections
      users/ account/ cluster/ settings/   one page each plus its components
    components/
      ui/                 library components (shadcn, Magic UI), never edit
      layout/             app shell: sidebar, header, page header, cluster card
      common/             small shared pieces: copy button, status badge, usage bar, metric tile, code editor, egg icon, job list,
                          glow card (Magic UI), file drop zone, query state (loading/error/empty), confirm dialog,
                          user avatar
    lib/                  api.ts (HTTP client), queries/ (every query and mutation hook, one file per area),
                          types/ (api.gen.ts generated from the Go structs by make docs; one file per area on top),
                          formatting, notify (toasts), validation (form errors)
    hooks/                use-mobile (library), use-resolved-theme, use-auth, use-draft (form state), use-before-unload,
                          use-branding-head
scripts/uninstall.sh      cluster cleanup
charts/kubedactyl/        Helm chart (published as OCI artifact); Dockerfile and VERSION in the root
scripts/imagebuild/       daemonless image build and push (make image / push-image)
.github/workflows/        GitHub Actions build (make test lint image; run by hand: raise VERSION, push-image push-chart, release)
```

## API

Swagger UI: **http://localhost:8080/swagger/** (also linked in the sidebar; administrators can turn it off in
*Settings → API documentation*; `/swagger/` then answers 404, the API keeps working). Websocket
`/api/servers/{name}/ws` uses JSON events: `{"event":"console output","args":["…"]}`,
send `{"event":"send command","args":["say hi"]}` or `{"event":"set state","args":["restart"]}`.

All endpoints except `/auth/login`, `/auth/logout`, `/auth/invite`, `/setup`, `/branding`, `/legal`, `/health`
and `/info` need a session or an API token; the Swagger UI has an *Authorize* button for `Bearer <token>`.

`kubectl` works too: `kubectl -n kubedactyl get eggs,users.kubedactyl.io` and
`kubectl get gameservers.kubedactyl.io -A` (short names `kgs`, `kuser`, `kinvite`, `kdsettings`).
