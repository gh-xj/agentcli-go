#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

fail=0

say() {
  printf '%s\n' "$*"
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

search_lines() {
  local pattern="$1"
  local input="$2"
  if has_cmd rg; then
    printf '%s\n' "$input" | rg -n "$pattern"
  else
    printf '%s\n' "$input" | grep -En "$pattern"
  fi
}

search_repo_content() {
  local pattern="$1"
  if has_cmd rg; then
    rg -n "$pattern" . --glob '!.git'
  else
    grep -RInE --exclude-dir=.git "$pattern" .
  fi
}

say "[hygiene] checking tracked junk file paths"
JUNK_RE='(\.DS_Store|Thumbs\.db|(^|/)node_modules/|(^|/)dist/|(^|/)build/|(^|/)coverage/|(^|/)__pycache__/|(^|/)\.pytest_cache/|(^|/)\.idea/|(^|/)\.vscode/|\.log$|\.tmp$|\.swp$|\.swo$|\.bak$|\.orig$)'
if search_lines "$JUNK_RE" "$(git ls-files)" >/tmp/agentcli_hygiene_junk.txt; then
  say "[hygiene] ERROR: junk-like tracked paths found:"
  cat /tmp/agentcli_hygiene_junk.txt
  fail=1
fi

say "[hygiene] checking tracked sensitive file names"
SENSITIVE_RE='(\.pem$|\.key$|id_rsa$|id_ed25519$|\.p12$|\.jks$|\.keystore$|\.p8$|\.mobileprovision$|\.env$|\.env\.)'
if search_lines "$SENSITIVE_RE" "$(git ls-files)" >/tmp/agentcli_hygiene_sensitive_paths.txt; then
  say "[hygiene] ERROR: sensitive-like tracked paths found:"
  cat /tmp/agentcli_hygiene_sensitive_paths.txt
  fail=1
fi

say "[hygiene] checking for common leaked token/private-key signatures"
CONTENT_RE='(AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z\-_]{35}|ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{80,}|xox[baprs]-[A-Za-z0-9-]{10,}|-----BEGIN (RSA|OPENSSH|EC|DSA|PRIVATE) KEY-----)'
if search_repo_content "$CONTENT_RE" >/tmp/agentcli_hygiene_sensitive_content.txt; then
  say "[hygiene] ERROR: suspicious secret content found:"
  cat /tmp/agentcli_hygiene_sensitive_content.txt
  fail=1
fi

say "[hygiene] checking history for junk artifacts"
if search_lines '(\.DS_Store|Thumbs\.db)$' "$(git rev-list --objects --all)" >/tmp/agentcli_hygiene_history_junk.txt; then
  say "[hygiene] ERROR: junk artifacts still exist in history:"
  cat /tmp/agentcli_hygiene_history_junk.txt
  fail=1
fi

say "[hygiene] checking tracked files for oversized blobs (>1MB)"
if has_cmd awk; then
  while IFS= read -r tracked_path; do
    [ -f "$tracked_path" ] || continue
    size_bytes="$(wc -c <"$tracked_path")"
    if [ "$size_bytes" -gt 1048576 ]; then
      printf '%s:%s\n' "$size_bytes" "$tracked_path"
    fi
  done < <(git ls-files) >/tmp/agentcli_hygiene_large_tracked.txt
  if [ -s /tmp/agentcli_hygiene_large_tracked.txt ]; then
    say "[hygiene] ERROR: oversized tracked files found:"
    cat /tmp/agentcli_hygiene_large_tracked.txt
    fail=1
  fi
fi

say "[hygiene] checking history for oversized blobs (>1MB)"
if has_cmd awk; then
  git rev-list --objects --all \
    | git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(rest)' \
    | awk '$1=="blob" && $3>1048576 {print NR ":" $0}' >/tmp/agentcli_hygiene_large_blobs.txt
  if [ -s /tmp/agentcli_hygiene_large_blobs.txt ]; then
    say "[hygiene] WARN: large blobs remain in history and may require a separate cleanup pass:"
    cat /tmp/agentcli_hygiene_large_blobs.txt
  fi
fi

rm -f /tmp/agentcli_hygiene_junk.txt \
  /tmp/agentcli_hygiene_sensitive_paths.txt \
  /tmp/agentcli_hygiene_sensitive_content.txt \
  /tmp/agentcli_hygiene_history_junk.txt \
  /tmp/agentcli_hygiene_large_tracked.txt \
  /tmp/agentcli_hygiene_large_blobs.txt

if [ "$fail" -ne 0 ]; then
  say "[hygiene] FAILED"
  exit 1
fi

say "[hygiene] passed"
