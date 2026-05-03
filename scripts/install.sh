#!/usr/bin/env bash
# sspt one-line installer (Linux).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/NerdyGriffin/steam-switch-pro-templates/main/scripts/install.sh | bash
#
# What it does:
#   1. Fetches the latest release from GitHub
#   2. Downloads sspt-linux-amd64 to ~/.local/bin/sspt
#   3. Verifies the SHA256 against the published SHA256SUMS file
#   4. NOTE: `sspt install` is not yet implemented for Linux (Phase 3).
#      For now this script just places the binary; you can run `sspt apply`
#      manually or wire up your own systemd unit.
#
# All actions are user-scope; no sudo required.

set -euo pipefail

REPO='NerdyGriffin/steam-switch-pro-templates'
ASSET='sspt-linux-amd64'
DEST_DIR="${HOME}/.local/bin"
DEST="${DEST_DIR}/sspt"

echo "==> Fetching latest release info from ${REPO} ..."
api=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")
tag=$(printf '%s' "${api}" | grep -oE '"tag_name":\s*"[^"]+"' | head -1 | sed -E 's/.*"([^"]+)"/\1/')
echo "    latest tag: ${tag}"

asset_url=$(printf '%s' "${api}" | grep -oE '"browser_download_url":\s*"[^"]+'"${ASSET}"'"' | head -1 | sed -E 's/.*"([^"]+)"/\1/')
sums_url=$(printf '%s' "${api}"  | grep -oE '"browser_download_url":\s*"[^"]+SHA256SUMS"'                | head -1 | sed -E 's/.*"([^"]+)"/\1/')
[ -n "${asset_url}" ] || { echo "release ${tag} has no asset named ${ASSET}" >&2; exit 1; }
[ -n "${sums_url}" ]  || { echo "release ${tag} has no SHA256SUMS asset"     >&2; exit 1; }

tmp=$(mktemp -d -t sspt-install-XXXXXX)
trap 'rm -rf "${tmp}"' EXIT

echo "==> Downloading ${ASSET} ..."
curl -fsSL "${asset_url}" -o "${tmp}/${ASSET}"
curl -fsSL "${sums_url}"  -o "${tmp}/SHA256SUMS"

echo "==> Verifying SHA256 ..."
expected=$(grep -E "[[:space:]]${ASSET}\$" "${tmp}/SHA256SUMS" | awk '{print $1}')
[ -n "${expected}" ] || { echo "SHA256SUMS does not list ${ASSET}" >&2; exit 1; }
actual=$(sha256sum "${tmp}/${ASSET}" | awk '{print $1}')
if [ "${expected}" != "${actual}" ]; then
  echo "checksum mismatch for ${ASSET}" >&2
  echo "  expected: ${expected}" >&2
  echo "  actual:   ${actual}"   >&2
  exit 1
fi
echo "    checksum OK (${actual})"

mkdir -p "${DEST_DIR}"
install -m 0755 "${tmp}/${ASSET}" "${DEST}"
echo "==> Installed to ${DEST}"

case ":${PATH}:" in
  *":${DEST_DIR}:"*) ;;
  *) echo "    note: ${DEST_DIR} is not on PATH; add it to your shell rc to invoke 'sspt' directly" ;;
esac

echo ""
echo "==> Done. Try 'sspt status' to inspect state."
echo "    Service registration ('sspt install') is not yet implemented on Linux"
echo "    (Phase 3). For now run 'sspt apply' manually or wire up a systemd unit."
