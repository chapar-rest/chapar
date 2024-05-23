#!/bin/bash

set -euo pipefail

BASEDIR=$(dirname "$(realpath "$0")")

PREFIX=${PREFIX:-/usr/local}
BIN_DIR=$PREFIX/bin
APP_DIR=$PREFIX/share/applications
ICON_DIR=$PREFIX/share/icons

if [ ! -d "$BIN_DIR" ]; then
    mkdir -pv "$BIN_DIR"
fi

install "$BASEDIR/chapar" "$PREFIX/bin/chapar" && echo "$PREFIX/bin/chapar"

if [ ! -d "$APP_DIR" ]; then
    mkdir -pv "$APP_DIR"
fi
# Update icon path in desktop entry
sed -i "s#{ICON_PATH}#$ICON_DIR#" "$BASEDIR/desktop-assets/chapar.desktop"
cp -v "$BASEDIR/desktop-assets/chapar.desktop" "$PREFIX/share/applications/"

if [ ! -d "$ICON_DIR" ]; then
    mkdir -pv "$ICON_DIR"
fi
cp -v "$BASEDIR/appicon.png" "$PREFIX/share/icons/chapar.png"
