#!/usr/bin/env bash
#
# RESMA Installer entrypoint — roda install.sh, upgrade.sh ou uninstall.sh baseado em MODE.
set -euo pipefail

MODE="${MODE:-install}"

case "$MODE" in
  install)
    exec bash /install/install.sh
    ;;
  upgrade)
    exec bash /install/upgrade.sh
    ;;
  uninstall)
    exec bash /install/uninstall.sh
    ;;
  *)
    echo "ERROR: Unknown MODE='$MODE'. Use 'install', 'upgrade', or 'uninstall'."
    exit 1
    ;;
esac
