#!/bin/sh
# Installs the Cal binary, icon and launcher entry for the current user.
# Usage: ./install.sh [--system]   (default installs under ~/.local)
set -eu

if [ "${1:-}" = "--system" ]; then
  BINDIR=/usr/local/bin
  ICONDIR=/usr/local/share/icons/hicolor/512x512/apps
  DESKTOPDIR=/usr/local/share/applications
else
  BINDIR="$HOME/.local/bin"
  ICONDIR="$HOME/.local/share/icons/hicolor/512x512/apps"
  DESKTOPDIR="$HOME/.local/share/applications"
fi

mkdir -p "$BINDIR" "$ICONDIR" "$DESKTOPDIR"
install -m755 cal "$BINDIR/cal"
install -m644 cal.png "$ICONDIR/cal.png"
install -m644 cal.desktop "$DESKTOPDIR/cal.desktop"

echo "Cal installed to $BINDIR/cal"
echo "Run it from your app launcher or with: cal"
