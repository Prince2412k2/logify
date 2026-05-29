#!/usr/bin/env bash
# Removes the logify binary and (optionally) the user config.
set -euo pipefail

green() { printf '\033[32m%s\033[0m' "$1"; }
amber() { printf '\033[33m%s\033[0m' "$1"; }
say()   { printf '%s\n' "$@" >&2; }

removed=0
for p in "${LOGIFY_PREFIX:-}" "$HOME/.local/bin/logify" "/usr/local/bin/logify"; do
  [ -z "$p" ] && continue
  [ -d "$p" ] && p="$p/logify"
  if [ -f "$p" ]; then
    rm -f "$p" && say "$(green "✓") removed $p"
    removed=1
  fi
done
[ "$removed" -eq 0 ] && say "no logify binary found in the usual places"

cfg="${XDG_CONFIG_HOME:-$HOME/.config}/logify"
if [ -d "$cfg" ]; then
  say ""
  say "$(amber "?")  config still exists at $cfg"
  say "    remove it too?  rm -rf \"$cfg\""
fi
