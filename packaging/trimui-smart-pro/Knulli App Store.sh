#!/bin/bash
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$SCRIPT_DIR/knulli-app-store"
LOG_DIR="/userdata/system/logs"
LOG_FILE="$LOG_DIR/knulli-app-store.log"

mkdir -p "$LOG_DIR"
cd "$APP_DIR" || exit 1

if [ ! -x ./knulli-app-ui ] || [ ! -r ./catalog-index.json ]; then
  printf '%s\n' "Knulli App Store installation is incomplete." >>"$LOG_FILE"
  exit 1
fi

exec ./knulli-app-ui -catalog ./catalog-index.json >>"$LOG_FILE" 2>&1
