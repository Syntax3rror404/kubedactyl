# Kubedactyl image: frontend build → Go build (frontend embedded) → Alpine.
# Build: docker build -t <registry>/kubedactyl:<tag> .   (see docs/installation.md)

FROM node:26.10.0-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# The npm table of THIRD_PARTY_NOTICES.md is checked against the bundle (scripts/licenses-plugin.ts).
COPY THIRD_PARTY_NOTICES.md /src/
# Writes to ../backend/web/dist (vite.config.ts)
RUN npm run build

FROM golang:1.27.1-alpine AS backend
# docker build --build-arg VERSION=$(cat VERSION) .
ARG VERSION=dev
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/backend/web/dist ./web/dist
COPY LICENSE THIRD_PARTY_NOTICES.md /src/
RUN go run ./tools/licenses -root /src -dist web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.appVersion=${VERSION}" -o /out/kubedactyl .

# Alpine (not Debian): small, ships the CA bundle for HTTPS (egg import from URLs).
FROM alpine:3.24.2
RUN addgroup -S -g 65532 kubedactyl && adduser -S -D -H -u 65532 -G kubedactyl kubedactyl
COPY --from=backend /out/kubedactyl /kubedactyl
COPY --from=backend /src/backend/web/dist/third-party-licenses.md /usr/share/licenses/kubedactyl/THIRD_PARTY_LICENSES.md
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/kubedactyl"]
