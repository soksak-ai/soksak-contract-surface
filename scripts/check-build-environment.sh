#!/bin/sh
set -eu

[ "$#" -eq 0 ] || { echo 'BUILD_DECLARATION_INVALID: usage: check-build-environment.sh' >&2; exit 78; }
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
go_expected=$(awk '$1 == "go" { value="go" $2; count++ } END { if (count == 1) print value; else exit 1 }' "$root/go.mod" 2>/dev/null || true)
go_actual=$(go env GOVERSION 2>/dev/null || true)
go_host_os=$(go env GOHOSTOS 2>/dev/null || true)
go_host_arch=$(go env GOHOSTARCH 2>/dev/null || true)
required_os=$go_host_os
required_arch=$go_host_arch
if [ "$(uname -s)" = Darwin ] && [ "$(sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then required_os=darwin; required_arch=arm64; fi
if [ "$go_actual" != "$go_expected" ] || [ "$go_host_os" != "$required_os" ] || [ "$go_host_arch" != "$required_arch" ]; then
  printf 'TOOLCHAIN_MISMATCH: expected go=%s runtime=%s/%s; actual go=%s runtime=%s/%s\n' \
    "${go_expected:-missing}" "$required_os" "$required_arch" "${go_actual:-missing}" "${go_host_os:-unknown}" "${go_host_arch:-unknown}" >&2
  exit 78
fi
rust_expected=$(sed -n 's/^channel = "\([^"]*\)"$/\1/p' "$root/rust-toolchain.toml")
rust_actual=$(rustc --version 2>/dev/null | awk '{print $2}' || true)
rust_host=$(rustc -vV 2>/dev/null | sed -n 's/^host: //p' || true)
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) rust_required_host=aarch64-apple-darwin ;;
  Darwin-x86_64) if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then rust_required_host=aarch64-apple-darwin; else rust_required_host=x86_64-apple-darwin; fi ;;
  Linux-aarch64|Linux-arm64) rust_required_host=aarch64-unknown-linux-gnu ;;
  Linux-x86_64) rust_required_host=x86_64-unknown-linux-gnu ;;
  MINGW*-x86_64|MSYS*-x86_64|CYGWIN*-x86_64) rust_required_host=x86_64-pc-windows-msvc ;;
  *) echo 'TOOLCHAIN_MISMATCH: unsupported host' >&2; exit 78 ;;
esac
if [ -z "$rust_expected" ] || [ "$rust_actual" != "$rust_expected" ] || [ "$rust_host" != "$rust_required_host" ]; then
  printf 'TOOLCHAIN_MISMATCH: expected rust=%s host=%s; actual rust=%s host=%s\n' \
    "${rust_expected:-missing}" "$rust_required_host" "${rust_actual:-missing}" "${rust_host:-unknown}" >&2
  exit 78
fi
printf 'BUILD_ENVIRONMENT_READY go=%s runtime=%s/%s rust=%s\n' "$go_actual" "$go_host_os" "$go_host_arch" "$rust_actual"
