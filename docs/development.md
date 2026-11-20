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
| `make dev` | Vite (5173) + Go API (8080) with `-tags dev`; Vite proxies `/api` (incl. websocket) |
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
folder and stops it again. Cluster integration test: `KUBE_IT_CONTEXT=<context> go test ./base/kube/ -run Attach -v`
(creates and deletes the namespace `kubedactyl-attachtest`).

Code hygiene (no warnings expected): `make lint` runs
[golangci-lint](https://golangci-lint.run) with `backend/.golangci.yml` (go vet, staticcheck, errcheck, unused code,
lines of at most 120 characters, functions of at most 60 lines / 40 statements (tests excepted), no cognitive
complexity above 20 (except the ported config file parsers in `internal/configfile`), gofmt and goimports), `deadcode`
(unreachable functions; `internal/testutil` is only used by tests) and the frontend checks
`npm run format:check && npx tsc -b && npx oxlint && npx knip` (unused files, exports and dependencies; library files are
ignored via `knip.json`). golangci-lint and deadcode run in fixed versions (`GOLANGCI`, `DEADCODE` in the `Makefile`;
raised by Renovate). `make fmt` rewrites the formatting (Go: gofmt + goimports, frontend: Prettier); long lines are
wrapped by hand.

## Dependency updates

Renovate (`.github/renovate.json5`) opens one pull request a week with every minor and patch update, one per major
update and one that refreshes `package-lock.json`; its dependency dashboard issue lists them all. The CI checks each
of them (tests, lint, image build); a green pull request needs nothing but the merge. The license tables in
`THIRD_PARTY_NOTICES.md` name no versions, so updates pass; a new package fails the build until its license is
checked and its row added. Library components are not part of it, nor `recharts`, whose version the shadcn registry
names: both come with `make ui-update`.

## Library components and updates

Files that come from a library are **never edited by hand**, so they can be updated at any time:

| Files | Source | Update |
|-------|--------|--------|
| `frontend/src/components/ui/*` (except the two below) | shadcn registry | `make ui-update` |
| `frontend/src/components/ui/magic-card.tsx`, `light-rays.tsx`, `particles.tsx`, `shine-border.tsx` | [Magic UI](https://magicui.design) registry (`@magicui/…`, MIT) | `make ui-update` |
| `frontend/src/components/theme-provider.tsx` | code block of the [shadcn Vite dark mode docs](https://ui.shadcn.com/docs/dark-mode/vite) | `make ui-update` |
| `frontend/src/hooks/use-mobile.ts`, `frontend/src/lib/utils.ts` | installed by shadcn together with `sidebar` / `init` | with the components |

The list lives in `frontend/ui-components.json`. `make ui-update` runs `frontend/scripts/update-ui.mjs`:
it re-adds every entry with `--overwrite` and fetches the docs file. The files stay exactly as the
sources deliver them (some keep a `"use client"` directive, which Vite ignores), and their dependencies get the
versions the registry items name (`chart` pins `recharts`; a newer pin arrives with the next update). The CLI lists
the files it updated (identical ones are skipped). Unused code inside
library files is not touched: TypeScript only
checks types, unused code is reported by oxlint (`npm run lint`, part of `npm run build`), which
ignores the library files.

Running the update twice produces no diff. Earlier hand edits were replaced by wrappers:

| Former edit | Now |
|-------------|-----|
| `ui/sonner.tsx` read the theme from `next-themes` → changed to our provider | original file; `base/components/app-toaster.tsx` passes the theme of our `ThemeProvider` (the registry Toaster spreads props after `theme`), `next-themes` stays installed as its dependency |
| unused imports in registry files | original files; unused-code checks moved from `tsc` to oxlint, which skips library files |
| `mode-toggle.tsx` from the dark mode docs | our own `base/components/mode-toggle.tsx`: the docs show only the Radix version |
| `sidebar-07` block demo files (`team-switcher`, `nav-*`) | built after in `base/navigation/` (the same in every app; each app fills `app/use-navigation.ts`); blocks are starting points, not updated |

When adding a registry component: install it with the CLI (`npx shadcn@latest add <name>` or
`@magicui/<name>`) **and** add it to `ui-components.json`. Only these two registries are used (both MIT);
app specific looks go into wrappers, e.g. `base/components/glow-card.tsx` (Magic UI card in the brand colors)
and `base/components/file-dropzone.tsx` (own drop zone on `react-dropzone`).

Status colors are theme tokens next to shadcn's `destructive` in `src/index.css`: `success`, `warning`, `pending`
(something is changing, like a server starting) and `info` (`text-success`, `bg-pending/10`, ...). The server phases
have one table, `phaseStatus` in `lib/format.ts`, for the badge and the sidebar. Fixed Tailwind colors are only for decoration (brand gradient, egg icons,
file type icons, the terminal).

## Project layout

Rule of thumb: **one folder per topic**. In the backend every package under `internal/` has one job;
file name prefixes group the files of a package (`servers_*.go`, `files_*.go`, `gameserver_*.go`).
In the frontend every feature has its own folder with its pages (`*-page.tsx`) and the components,
hooks and helpers only it uses; shared pieces live in `components/` and `lib/`, the pieces of every app in `base/`.

```
backend/
  main.go · wiring.go     start-up with the base parts: flags, logging, prepare the cluster · build controllers,
                          services and API, HTTP server (main_test.go: the Swagger doc matches the routes)
  app/                    the app's side of the base: config.go (name, ID, prefixes, logo, brand colors, version
                          modules) · views.go (the app's fields in base answers)
  base/                   the same in every app, treated like a library (change it here, copy the folder):
    api/                  HTTP API of every app: api.go (API, App, base routes) · errors.go (a.Fail, status mapping)
                            · auth*.go (middleware, login, account, setup, single sign-on) · users.go · invites.go
                            · settings.go (settings, legal texts, branding, request rates) · upgrade.go (and versions)
    auth/                 passwords, sessions, device cookie, signed values, API tokens, login limiter, single sign-on
    users/ · settings/    accounts, invites, reset links, last-admin rule · panel settings (the app adds fields: App)
    bootstrap/            start-up: scheme, CRDs, panel namespace, session key, first admin / setup token
    httpserver/           router (API, API document, SPA), security headers/CSP, body limit, trusted proxies, health/info
    kube/                 connection, clients, exec / attach, request limit, TTL cache, CreateOrPatch, volume helpers
    selfupgrade/          panel updates: registry check, upgrade job, `<app> upgrade` (Helm SDK)
    validation/ · checks/ field validation errors (422 with messages per field) · check results (health, diagnostics)
    cli/                  flags of every panel, start-up logo (colored on a terminal only), slog handler without
                          client-go noise
    tools/licenses/       build step: third-party-licenses.md
  api/v1alpha1/           CRD types: Egg, GameServer, User, PanelSettings (kubebuilder markers)
  config/crd/             generated CRD manifests (make generate), embedded and applied on start
  docs/                   generated Swagger (make docs)
  web/                    embedded frontend build
  internal/
    httpapi/              REST + websocket handlers of the app (embeds base/api), grouped by prefix:
                            api.go (routes) · errors.go (the app's errors) · users.go · settings.go (the app's parts of
                            base answers, storage classes, pools)
                            servers*.go (list/get/delete, create, update, access rules, power/reinstall/suspend, console websocket, stats, backups)
                            files_*.go (read, write, session) · eggs.go · cluster.go
    appusers/ · appsettings/  what this app adds to accounts (namespaces, subusers) and settings (storage, pools, ...)
    controller/           reconcilers: gameserver*.go (entry, children, install, game, cleanup of finished pods)
                            · files_reaper.go (idle files pods) · user*.go (user namespaces, network policy)
    gameserver/           Kubernetes objects of a server (pods, PVC, service, placement, resources), validation, pre-start steps
    serverctl/            power actions, console commands, reinstall, suspend, transfer, migrate (shared by API and schedules)
    schedule/             cron schedules of servers
    eggstore/             storing eggs (create, import, update from URL, delete)
    console/              output hub (history, subscribers), log following, "done" detection
    files/                file manager via exec in the files pod, activity tracker, background jobs, backups, URL downloads
    clusterinfo/          cluster page: nodes, hardware probes, identity, permission checks
    egg/ · configfile/    egg parsing, validation, export (PLCN_v3 / PTDL_v2) · config file parsers of eggs
    tenancy/              user namespaces, installation scoping
frontend/
  ui-components.json      library component manifest (make ui-update)
  scripts/ · e2e/         UI update scripts · browser smoke test (npm run e2e)
  public/theme-init.js    theme before first paint (external because of the CSP)
  src/
    main.tsx              entry: providers
    app/                  router (URL → page); the app's side of the base: use-navigation.ts (sidebar content),
                          config.ts (names, brand, roles, footer links, request rates, breadcrumb names),
                          users.tsx (user table columns, account cards), settings.ts (the app's settings and cards)
    base/                 the same in every app, treated like a library (change it in one app, copy the folder):
      components/         callout, confirm dialog, copy button, status icons, usage bar, metric tile, code editor,
                          glow card (Magic UI), file drop zone, query state (loading/error/empty), brand mark, ...
      layout/             app layout (a route handle `wide: true` gives a page the whole width), header with
                          breadcrumbs, footer with request rates, page header, error pages
      navigation/         sidebar, its header (short diagnosis), search (⌘K), user menu
      hooks/              use-auth, use-draft (form state), use-throttle, use-retained, use-branding-head ...
      lib/                api.ts (request, ApiError, the base calls) · queries/ (auth, users, settings, keys,
                          mutation) · types.ts (base API types) · format, notify, validation, legal
      features/           the pages of backend/base/api: auth (login, setup, invite, change password), account,
                          users, settings (base cards), licenses
      types.ts            what an app gives the base, the states its components show
    features/
      dashboard/          dashboard-page · server-card
      servers/            server-layout, server-new-page and one page per server tab (URL /servers/:server/<tab> →
                          <tab>-page.tsx): console, files, startup, schedules, backups, settings
                            · components (server form, power, console, stats, resources, ports, placement, file-* for
                              the file manager, …) · hooks · egg-features (eula, java_version, …) · lib
      eggs/               list, detail, edit (tabs) and new (wizard) pages; components/editor = shared form sections
      cluster/            cluster page plus its components
      account/ settings/  the app's cards on the base pages (subusers; notice, address, pools, egg libraries, ...)
    components/
      ui/                 library components (shadcn, Magic UI), never edit
      common/             pieces only this app has: egg icon, server status badge, access select, effects
    lib/                  api.ts (the app's calls on top of baseApi), queries/ (the app's query and mutation hooks,
                          one file per area), types/ (api.gen.ts generated from the Go structs by make docs; one file
                          per area on top), format (servers, pools, phases)
    hooks/                use-mobile (library)
scripts/uninstall.sh      cluster cleanup
charts/kubedactyl/        Helm chart (published as OCI artifact); Dockerfile and VERSION in the root
scripts/imagebuild/       daemonless image build and push (make image / push-image)
.github/workflows/        GitHub Actions build (make test lint image; run by hand: raise VERSION, push-image push-chart, release)
```

## Starting a new app from the base

The simplest start is a copy of an existing app with its features removed: everything the base expects is then
already in place. The base needs these parts from the app:

- **Backend:** the Go module is named `app` (the base imports `app/app` and `app/api/v1alpha1`).
  `app/config.go` (names, prefixes, brand) and `app/views.go` (the app's fields in base answers).
  The CRD types `User`, `Invite` and `PanelSettings` in `api/v1alpha1` with the fields the base reads (roles and the
  app's own settings may differ), then `make generate`. `main.go` + `wiring.go` start the panel with `base/cli`,
  `base/bootstrap` and `base/httpserver`; the app's `httpapi.API` embeds `base/api.API`, implements `api.App` and
  adds its routes to the `Signed` and `Admin` groups that `API.Register` returns.
- **Frontend:** `app/config.ts`, `app/use-navigation.ts`, `app/users.tsx`, `app/settings.ts` and `app/router.tsx`;
  the library components in `ui-components.json` (`make ui-update`), `components/theme-provider.tsx` and
  `lib/utils.ts`; the color tokens and fonts in `src/index.css`; `vite.config.ts` (`__FRONTEND_VERSIONS__`, the
  licenses plugin from `scripts/licenses-plugin.ts`).
- **Build and chart:** the Makefile targets (`docs`, `licenses`, `check-base`), `THIRD_PARTY_NOTICES.md` and the
  chart templates `rbac.yaml`, `admission-policy.yaml` and `self-upgrade.yaml` (names from `app.ID`).

Then set `BASE_PEER` in `local.mk` to this repository and check with `make check-base` that both base folders match.

## Comment style

English, lines <= 120 columns (gofmt tabs, Prettier in the frontend). A comment must earn its line: fewer, better
placed comments. Explain the why, the order and the intent; never narrate what the next line does.

### Structure

1. **The doc comment is the manual.** Exported types, functions, components and hooks start with a doc comment
   (Go: `// Name ...`, TS: `/** ... */`), usually one sentence, at most ~10 lines: what it is, how it behaves
   (rules, not history), how to use it (who calls it, what it needs), limits.
2. **Where it runs.** Files are split by responsibility, so most run in one place only. A file that mixes places
   gets one chapter per place, in the same order as in its test file:
   ```go
   // ---------------------------------------------------------------------------------------------------------------
   // Controller
   ```
   Chapters: `Request handler`, `Controller`, `Background` (runner, reaper, updater goroutines), `Pod script`.
   Data handed between them through the CRD sits in its own block: `// Handler -> controller (spec)`,
   `// Controller -> UI (status)`. Shared memory says what guards it ("guarded by mu").
3. **One-line why at the exception.** Inside a function, comment only what is not obvious: the reason, the
   exception, the ordering constraint, which cache or reader is used and why.

### Complex logic

4. **Overview first, steps numbered.** A function or flow with three or more stages (reconcilers, migration,
   transfer, upgrade) starts with an overview: one sentence of goal, then the stages `1. ... 2. ... 3. ...`. In
   the body each stage starts with the matching `// 2. Title` (plus a short why). Reading only these lines gives
   the flow.
   ```go
   // reconcileMigration moves the server files while spec and status name different storage classes.
   //  1. Stop: the game pod is removed first, a running game would write during the copy.
   //  2. Copy: a pod copies the old volume into a new claim and compares count and bytes.
   //  3. Switch: both volumes are retained and the step is saved, then the claims are deleted.
   //  4. Finish: the old volume is deleted once the new claim is bound.
   ```
5. **Fallback chains are numbered in priority order**, the same way as stages.
6. **Order and timing:** "Before X, because ...", "Last: ...", "Saved before anything is deleted, so ...".
7. **Cluster intent = what it guarantees + what goes wrong without it.** "Retained first, so a restart here
   loses no data."
8. **Quantities: define once.** Give each non-obvious value its meaning, range and unit once ("share of the
   limit, 0..1", "seconds").
9. **Edge cases as rules, not stories.** Write what the code guarantees, not the bug that happened. If the story
   matters, name the test that guards it.
10. **Labels:** `Workaround:` (Kubernetes, BusyBox, proxy or browser behaviour, the version, when to remove),
    `Fail-safe:`, `Experimental:`.

### API fields and UI hints

11. **CRD and API field comments are written for the administrator or API user**: they become the CRD
    description, the Swagger docs and the frontend types. The first sentence says what it does for the server.
    Bools describe the on state. Skip it only where the name and the unit say everything.
12. **Shape, at most ~3 lines:** `<What it does>. <Hint: higher = ..., lower = ...>. <Off / default / limit>.`
    Fixed phrases: "0 = off.", "Empty: uses X.", "Optional.", "Ignored when X.", "Only with X.", "Admins only.".
13. **Units in the name or in parentheses** at the end of the first sentence ("(seconds)", "(MiB)"); ranges as
    `+kubebuilder:validation:Minimum` / `Maximum` where they exist.
14. **UI hints** (descriptions, notices) are one short sentence, two at most; details go to `docs/`.

### What never goes into the code

15. **Pterodactyl and Pelican only where eggs are concerned.** Describe panel behaviour and its reason, never
    "like X". No internal names (clusters, domains, IPs, people).
16. **No development log:** no user quotes, measurements or bug stories in comments, API docs or UI texts. They
    belong in commit messages and tests.
17. **Plain sentences:** present tense, one idea per sentence, sentence case, at most one colon per sentence, no
    en or em dashes.

### Budget

Comments should not grow the code. New overviews and field comments are paid for by removing history, narration
and repeated explanations. Existing code follows this style when it is changed anyway, not in a sweep.

## API

API reference: **http://localhost:5173/api-docs** (Scalar, reads `/api/openapi.json`; also linked at the bottom of the
sidebar; administrators can turn it off in *Settings → Security*; `/api/openapi.json` then answers 404, the API keeps
working). Request bodies are strict JSON: a field the endpoint does not know is refused with 400 ("unknown field").
Websocket `/api/servers/{name}/ws` uses JSON events: `{"event":"console output","args":["…"]}`,
send `{"event":"send command","args":["say hi"]}` or `{"event":"set state","args":["restart"]}`.

All endpoints except `/auth/login`, `/auth/logout`, `/auth/invite`, `/setup`, `/branding`, `/legal`, `/health`,
`/info` and `/openapi.json` need a session or an API token; the API reference takes `Bearer <token>` under *Authentication*.

`kubectl` works too: `kubectl -n kubedactyl get eggs,users.kubedactyl.io` and
`kubectl get gameservers.kubedactyl.io -A` (short names `kgs`, `kuser`, `kinvite`, `kdsettings`).
