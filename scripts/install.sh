#!/usr/bin/env bash
# GIRL Installer: build, install framework agents, configure PATH
set -euo pipefail

GIRL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="$GIRL_DIR/girl"

echo "==> Building GIRL..."
cd "$GIRL_DIR"
go build -o "$BINARY" ./cmd/girl/
echo "    Built: $BINARY"

echo "==> Installing OpenCode agents..."
"$BINARY" install opencode

echo "==> Adding GIRL to PATH..."
case "${SHELL:-}" in
  *fish)
    FISH_CONFIG="${HOME}/.config/fish/config.fish"
    LINE="fish_add_path ${GIRL_DIR}"
    if [ -f "$FISH_CONFIG" ] && grep -qF "$LINE" "$FISH_CONFIG" 2>/dev/null; then
      echo "    Already in fish config"
    else
      echo "$LINE" >> "$FISH_CONFIG"
      echo "    Added to $FISH_CONFIG"
      fish -c "fish_add_path ${GIRL_DIR}" 2>/dev/null || true
    fi
    ;;
  *zsh)
    ZSHRC="${HOME}/.zshrc"
    LINE="export PATH=\"${GIRL_DIR}:\$PATH\""
    if [ -f "$ZSHRC" ] && grep -qF "GIRL_DIR" "$ZSHRC" 2>/dev/null; then
      echo "    Already in .zshrc"
    else
      printf "\n# GIRL\nexport PATH=\"%s:\$PATH\"\n" "$GIRL_DIR" >> "$ZSHRC"
      echo "    Added to $ZSHRC"
    fi
    ;;
  *bash)
    BASHRC="${HOME}/.bashrc"
    LINE="export PATH=\"${GIRL_DIR}:\$PATH\""
    if [ -f "$BASHRC" ] && grep -qF "GIRL_DIR" "$BASHRC" 2>/dev/null; then
      echo "    Already in .bashrc"
    else
      printf "\n# GIRL\nexport PATH=\"%s:\$PATH\"\n" "$GIRL_DIR" >> "$BASHRC"
      echo "    Added to $BASHRC"
    fi
    ;;
  *)
    echo "    Unknown shell $SHELL. Add to PATH manually:"
    echo "    export PATH=\"$GIRL_DIR:\$PATH\""
    ;;
esac

echo ""
echo "==> GIRL install complete."
echo "    Binary: $BINARY"
echo "    Run 'source ${SHELL/#*fish/config.fish}' or open new shell to pick up PATH."
echo "    Try 'girl analyze --help'"
