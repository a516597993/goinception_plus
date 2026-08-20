#!/usr/bin/env sh
set -eu
CONFIG_PATH="${1:-./config/config.toml}"
PORT="$(awk -F= '/^[[:space:]]*port[[:space:]]*=/{gsub(/[[:space:]]/,"",$2); print $2; exit}' "$CONFIG_PATH")"
PORT="${PORT:-4000}"
if command -v ss >/dev/null 2>&1 && ss -ltn "sport = :$PORT" | grep -q LISTEN; then
  echo "Port $PORT is already listening; stop the existing process before starting a new configuration." >&2
  exit 1
fi
exec ./bin/linux-amd64/goinception-plus -config "$CONFIG_PATH"
