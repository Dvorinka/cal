#!/bin/sh
# Cal — self-hosted planner. Downloads the compose stack, generates a .env,
# and brings it up.
#
#   curl -fsSL https://raw.githubusercontent.com/Dvorinka/cal/main/install.sh | sh
#
# Installs into ./cal (override: CAL_DIR=/opt/cal). Idempotent: re-running
# pulls fresh images and restarts; .env and the named volumes are never
# touched. From a git checkout, ./install.sh installs in place. Overrides:
#
#   PORT=9090 sh install.sh          # host port (default 8080, first run)
#   CAL_VERSION=v1.2.3 sh install.sh # pin image tag (default latest)
#   CAL_REF=main sh install.sh       # git ref for the downloaded compose file
set -eu

CAL_DIR="${CAL_DIR:-cal}"
PORT="${PORT:-8080}"
CAL_REF="${CAL_REF:-main}"
BASE="https://raw.githubusercontent.com/Dvorinka/cal/$CAL_REF"

command -v docker >/dev/null 2>&1 || {
  echo "error: docker not found — install Docker first: https://docs.docker.com/get-docker/" >&2
  exit 1
}
docker info >/dev/null 2>&1 || {
  echo "error: docker daemon not reachable (try: sudo usermod -aG docker $USER, then re-login)" >&2
  exit 1
}
docker compose version >/dev/null 2>&1 || {
  echo "error: docker compose plugin not found — https://docs.docker.com/compose/install/" >&2
  exit 1
}

# A compose file in cwd means a git checkout — install in place. Otherwise
# fetch it into CAL_DIR.
if [ -f docker-compose.yml ]; then
  CAL_DIR=.
else
  mkdir -p "$CAL_DIR"
  echo "→ fetching docker-compose.yml ($CAL_REF)"
  curl -fsSL "$BASE/docker-compose.yml" -o "$CAL_DIR/docker-compose.yml"
fi
cd "$CAL_DIR"

# .env carries the generated DB password and the host port; never overwritten,
# so re-runs reuse the same database credentials.
if [ ! -f .env ]; then
  if command -v openssl >/dev/null 2>&1; then
    PW="$(openssl rand -hex 24)"
  else
    PW="$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  fi
  printf 'POSTGRES_PASSWORD=%s\nPORT=%s\n' "$PW" "$PORT" > .env
  chmod 600 .env
  echo "→ generated .env"
else
  # .env wins on re-runs — health check uses the port it actually recorded.
  SAVED_PORT="$(sed -n 's/^PORT=//p' .env | tail -1)"
  PORT="${SAVED_PORT:-$PORT}"
fi

echo "→ pulling images"
docker compose pull

echo "→ starting stack"
docker compose up -d

echo "→ waiting for Cal to come up"
i=0
while [ "$i" -lt 120 ]; do
  if curl -fsS "http://localhost:$PORT/api/health" >/dev/null 2>&1; then
    break
  fi
  i=$((i + 1))
  sleep 1
done

cat <<EOF

Cal is running.

  UI + API   http://localhost:$PORT
  Install    $CAL_DIR (docker-compose.yml + .env)
  Volumes    cal-pg (database), cal-data (files) — back up both
  Stop       cd $CAL_DIR && docker compose down
  Upgrade    re-run this script

Open http://localhost:$PORT and create your account — the first one is yours.
EOF
