BIN := bin/kubedactyl

# Single source of the version: the VERSION file. It goes into the binary (API /info, footer,
# Swagger), the image tag and the chart version/appVersion.
VERSION := $(shell cat VERSION)
LDFLAGS := -s -w -X main.appVersion=$(VERSION)

# Kubernetes context used by `make dev` / `make run` / `make uninstall`: set it in local.mk (not versioned),
# e.g. `KUBE_CONTEXT := my-cluster`, or per call: make dev KUBE_CONTEXT=my-cluster
-include local.mk
export KUBE_CONTEXT

.PHONY: dev dev-backend dev-frontend generate docs build frontend licenses run check-context test lint fmt ui-update uninstall clean image push-image push-chart sync-version check-generated

# Commands that talk to a cluster need an explicit context (never silently the current one).
check-context:
	@test -n "$(KUBE_CONTEXT)" || { echo "KUBE_CONTEXT is not set: put e.g. 'KUBE_CONTEXT := my-cluster' into local.mk"; exit 1; }

# Dev: Vite (http://localhost:5173, hot reload) + Go API (:8080); Vite proxies /api and /swagger
dev:
	@trap 'kill 0' EXIT; \
	$(MAKE) dev-backend & \
	$(MAKE) dev-frontend & \
	wait

dev-backend: check-context generate docs
	cd backend && go run -tags dev -ldflags "-X main.appVersion=$(VERSION)-dev" .

dev-frontend:
	cd frontend && npm run dev

# Deepcopy functions and CRD manifests from the API types (backend/api/v1alpha1)
generate:
	cd backend && go tool controller-gen object paths=./api/... && \
		go tool controller-gen crd paths=./api/... output:crd:dir=./config/crd

# Swagger docs from the handler comments (backend/docs) and the frontend API types generated from them
# (frontend/src/lib/types/api.gen.ts). --requiredByDefault: fields without omitempty are always sent.
docs:
	cd backend && go tool swag fmt && \
		go tool swag init -g main.go -o docs --outputTypes go,json,yaml --parseInternal --parseDependency --parseDependencyLevel 1 --requiredByDefault --templateDelims "[[,]]" --quiet
	cd frontend && npm run gen:api

# Production: build the frontend and embed it into the Go binary
frontend:
	cd frontend && npm run build

# third-party-licenses.md (licenses page of the panel, shipped in the image): license texts of the dependencies
# listed in THIRD_PARTY_NOTICES.md; fails when the list and the build differ
licenses:
	cd backend && go run ./tools/licenses -root .. -dist web/dist

build: generate docs frontend licenses
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o ../$(BIN) .

run: check-context build
	./$(BIN)

# A fresh clone has no frontend build yet; go:embed needs at least one file in web/dist.
test: check-generated
	@test -n "$$(ls -A backend/web/dist 2>/dev/null)" || { mkdir -p backend/web/dist && echo "run make build for the web interface" > backend/web/dist/placeholder.txt; }
	cd backend && go test ./...

# Code rules: backend/.golangci.yml (line and function length, complexity, staticcheck, imports),
# unused code (deadcode; internal/testutil is only used by tests), and the frontend checks.
GOLANGCI := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
lint:
	cd backend && $(GOLANGCI) run ./...
	@cd backend && dead="$$(go run golang.org/x/tools/cmd/deadcode@latest ./... | grep -v internal/testutil/)"; \
		test -z "$$dead" || { echo "unused code:"; echo "$$dead"; exit 1; }
	cd frontend && npm run format:check && npx tsc -b && npx oxlint && npx knip

# Rewrites the formatting (gofmt + goimports, Prettier).
fmt:
	cd backend && $(GOLANGCI) fmt ./...
	cd frontend && npm run format

# Generated files (CRDs, deepcopy, Swagger, frontend API types) must match the code they come from.
check-generated:
	@bash scripts/check-generated.sh

# Re-install all shadcn / Magic UI / docs components listed in frontend/ui-components.json
ui-update:
	cd frontend && node scripts/update-ui.mjs && npx tsc -b

# Delete the namespace with all servers and data from the cluster (asks for confirmation).
# DELETE_CRDS=1 also removes the CRDs, YES=1 skips the prompt.
uninstall: check-context
	scripts/uninstall.sh

clean:
	rm -rf bin backend/web/dist

# ---- Release (linux/amd64) without a Docker daemon -------------------------------------
# The image mirrors the last Dockerfile stage (alpine + /kubedactyl + user 65532).
IMAGE ?= ghcr.io/syntax3rror404/kubedactyl
CHART_REPO ?= oci://ghcr.io/syntax3rror404/charts
GHCR_USER ?= Syntax3rror404

image: build
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o ../bin/kubedactyl-linux-amd64 .
	cd scripts/imagebuild && go run . build -bin ../../bin/kubedactyl-linux-amd64 -tag $(IMAGE):$(VERSION) \
		-out ../../bin/kubedactyl-$(VERSION)-amd64.tar -version $(VERSION) -source https://github.com/Syntax3rror404/kubedactyl \
		-licenses ../../backend/web/dist/third-party-licenses.md

# Needs `gh auth refresh -s write:packages`; the token is only passed as environment variable.
push-image: image
	cd scripts/imagebuild && GHCR_USER=$(GHCR_USER) GHCR_TOKEN=$$(gh auth token) go run . push -in ../../bin/kubedactyl-$(VERSION)-amd64.tar -tag $(IMAGE):$(VERSION)
	cd scripts/imagebuild && GHCR_USER=$(GHCR_USER) GHCR_TOKEN=$$(gh auth token) go run . push -in ../../bin/kubedactyl-$(VERSION)-amd64.tar -tag $(IMAGE):latest

# Writes VERSION into Chart.yaml (version and appVersion) and the chart README.
sync-version:
	sed -i.bak -e 's/^version: .*/version: $(VERSION)/' -e 's/^appVersion: .*/appVersion: "$(VERSION)"/' charts/kubedactyl/Chart.yaml
	sed -i.bak -e 's/--version [0-9][0-9.]*/--version $(VERSION)/' charts/kubedactyl/README.md README.md docs/installation.md
	rm -f charts/kubedactyl/Chart.yaml.bak charts/kubedactyl/README.md.bak README.md.bak docs/installation.md.bak

push-chart: sync-version
	helm package charts/kubedactyl -d bin
	@cfg=$$(mktemp); gh auth token | helm registry login ghcr.io --registry-config $$cfg --username $(GHCR_USER) --password-stdin && \
		helm push bin/kubedactyl-$(VERSION).tgz $(CHART_REPO) --registry-config $$cfg; rm -f $$cfg
