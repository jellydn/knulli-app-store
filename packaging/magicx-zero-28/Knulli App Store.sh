#!/bin/bash
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$SCRIPT_DIR/knulli-app-store"
LOG_DIR="/userdata/system/logs"
LOG_FILE="$LOG_DIR/knulli-app-store.log"

mkdir -p "$LOG_DIR"
cd "$APP_DIR" || exit 1

# Recovery can restore both the executable and its matching catalogue key.
# Run it in a separate CLI process before loading the UI binary.
./knulli-app recover >>"$LOG_FILE" 2>&1 || exit 1

if [ ! -x ./knulli-app-ui ] || [ ! -r ./catalog-index.json ] || [ ! -r ./catalog-index.json.sig ]; then
  printf '%s\n' "Knulli App Store installation is incomplete." >>"$LOG_FILE"
  exit 1
fi

# Metadata failure must not prevent access to app repair and diagnostics.
./knulli-app register-store-menu >>"$LOG_FILE" 2>&1 || true

# Ports uses direct SDL letters only when its Xbox-layout flag is present.
export STORE_CONTROLLER_LAYOUT=knulli
if [ -e /var/run/ports_xbox_layout.flag ]; then
  export STORE_CONTROLLER_LAYOUT=xbox
fi

exec ./knulli-app-ui -catalog ./catalog-index.json
