#!/bin/sh
# Cal — self-hosted planner. One container, one volume, nothing else.
#
#   curl -fsSL https://raw.githubusercontent.com/Dvorinka/cal/main/install.sh | sh
#
# Idempotent: re-running pulls the image and restarts the container; data in
# the cal-data volume is never touched. Override with env vars:
#
#   PORT=9090 sh install.sh          # host port (default 8080)
#   CAL_VERSION=v1.2.3 sh install.sh # pin a release tag (default latest)
#   CAL_NAME=mycal DATA_VOLUME=mycal-data sh install.sh
set -eu

PORT="${PORT:-8080}"
IMAGE="ghcr.io/dvorinka/cal:${CAL_VERSION:-latest}"
NAME="${CAL_NAME:-cal}"
DATA_VOLUME="${DATA_VOLUME:-cal-data}"

command -v docker >/dev/null 2>&1 || {
  echo "error: docker not found — install Docker first: https://docs.docker.com/get-docker/" >&2
  exit 1
}
docker info >/dev/null 2>&1 || {
  echo "error: docker daemon not reachable (try: sudo usermod -aG docker $USER, then re-login)" >&2
  exit 1
}

echo "→ pulling $IMAGE"
docker pull "$IMAGE"

# Replace a previous install cleanly; the volume survives.
docker rm -f "$NAME" >/dev/null 2>&1 || true

echo "→ starting $NAME on port $PORT (volume: $DATA_VOLUME)"
docker run -d --name "$NAME" \
  -p "$PORT:8080" \
  -v "$DATA_VOLUME:/data" \
  --restart unless-stopped \
  "$IMAGE" >/dev/null

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
  Data       docker volume: $DATA_VOLUME
  Stop       docker stop $NAME
  Upgrade    re-run this script

Open http://localhost:$PORT and create your account — the first one is yours.
EOF
