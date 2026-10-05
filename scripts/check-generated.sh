#!/usr/bin/env bash
# Fails when generated files are not up to date with the code they are generated from:
# CRD manifests and deepcopy code (make generate), Swagger docs and the frontend API types (make docs).
# It regenerates them in place, so after a failure the correct files are already there.
set -euo pipefail
cd "$(dirname "$0")/.."

generated=(
  backend/api/v1alpha1/zz_generated.deepcopy.go
  backend/config/crd
  backend/docs
  frontend/src/lib/types/api.gen.ts
)

before=$(mktemp -d)
trap 'rm -rf "$before"' EXIT
for path in "${generated[@]}"; do
  mkdir -p "$before/$(dirname "$path")"
  cp -R "$path" "$before/$path"
done

make --no-print-directory generate docs >/dev/null

stale=0
for path in "${generated[@]}"; do
  if ! diff -r -q "$before/$path" "$path" >/dev/null; then
    echo "stale: $path"
    stale=1
  fi
done
if [ "$stale" -ne 0 ]; then
  echo "Generated files were out of date and have been regenerated (make generate docs). Review and keep them." >&2
  exit 1
fi
echo "generated files are up to date"
