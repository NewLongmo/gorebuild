#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

SKIP_FRONTEND=false
SKIP_BACKEND_BUILD=false

while [ "$#" -gt 0 ]; do
  case "$1" in
    --skip-frontend)
      SKIP_FRONTEND=true
      ;;
    --skip-backend-build)
      SKIP_BACKEND_BUILD=true
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

mkdir -p "$ROOT_DIR/.cache/go-build"

echo "==> Smoke script syntax"
sh -n "$ROOT_DIR/scripts/smoke.sh"

echo "==> Backend tests"
(
  cd "$ROOT_DIR/backend"
  GOCACHE="$ROOT_DIR/.cache/go-build" go test ./...
)

if [ "$SKIP_BACKEND_BUILD" != "true" ]; then
  echo "==> Backend build"
  (
    cd "$ROOT_DIR/backend"
    GOCACHE="$ROOT_DIR/.cache/go-build" go build -o dw0rdwk-api ./cmd/server
  )
fi

if [ "$SKIP_FRONTEND" != "true" ]; then
  echo "==> Frontend build"
  (
    cd "$ROOT_DIR/frontend"
    npm run build
  )
fi

echo "Verification completed."
