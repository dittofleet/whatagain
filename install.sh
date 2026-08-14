#!/bin/sh
set -eu

REPO="dittofleet/whatagain"
DEST="${WHATAGAIN_INSTALL_DIR:-$HOME/.local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64) ARCH=arm64 ;;
  x86_64) ARCH=x64 ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

ASSET="whatagain-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

mkdir -p "$DEST"
# Staged inside the destination so the install is a same-filesystem
# rename. Downloading to $TMPDIR would make the final step a copy over the
# live binary, which an interruption could leave truncated.
TMP=$(mktemp "$DEST/.whatagain.XXXXXX")
trap 'rm -f "$TMP"' EXIT

echo "Downloading $URL..." >&2
curl -fsSL "$URL" -o "$TMP"
chmod 755 "$TMP"
mv "$TMP" "$DEST/whatagain"
echo "Installed whatagain to $DEST/whatagain" >&2

# There is nothing to configure: the store file is created on the first
# write, and an existing one (synced from another machine) is picked up
# as is.
CONFIG_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/whatagain/todo.json"
if [ -f "$CONFIG_FILE" ]; then
  echo "Using the existing store at $CONFIG_FILE" >&2
else
  echo "Items will be stored in $CONFIG_FILE" >&2
fi

case ":$PATH:" in
  *":$DEST:"*) ;;
  *) echo "Note: $DEST is not in \$PATH. Add it to your shell profile to use whatagain." >&2 ;;
esac
