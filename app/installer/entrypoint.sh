#!/usr/bin/env bash
#
# RESMA Installer entrypoint — roda install.sh ou uninstall.sh baseado em MODE.
set -euo pipefail

MODE="${MODE:-install}"

case "$MODE" in
  install)
    exec bash /install/install.sh
    ;;
  uninstall)
    exec bash /install/uninstall.sh
    ;;
  *)
    echo "ERROR: Unknown MODE='$MODE'. Use 'install' or 'uninstall'."
    exit 1
    ;;
esac
