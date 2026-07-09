#!/usr/bin/env bash
set -euo pipefail

install_dir="${INSTALL_DIR:-$HOME/.local/bin}"

case "${1:-}" in
  "")
    ;;
  --user)
    install_dir="$HOME/.local/bin"
    ;;
  --system)
    install_dir="/usr/local/bin"
    ;;
  -h|--help)
    cat <<'EOF'
Usage: ./install.sh [--user|--system]

Installs to ~/.local/bin by default.

Options:
  --user      Install to ~/.local/bin
  --system    Install to /usr/local/bin

Set INSTALL_DIR=/custom/bin to choose another install directory.
EOF
    exit 0
    ;;
  *)
    echo "azenv: unknown install option: $1" >&2
    echo "usage: ./install.sh [--user|--system]" >&2
    exit 1
    ;;
esac

go build -o azenv .
mkdir -p "$install_dir"
cp azenv "$install_dir/azenv"
echo "Installed azenv to $install_dir/azenv"
echo
echo 'For zsh, make sure ~/.local/bin is on PATH if using the default install:'
echo '  export PATH="$HOME/.local/bin:$PATH"'
echo
echo 'Then add this to your shell config:'
echo '  eval "$(azenv init zsh)"'
