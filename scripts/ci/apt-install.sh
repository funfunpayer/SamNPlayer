#!/usr/bin/env bash
# CI: install apt packages without hanging the job.
#
# On GitHub's Ubuntu runners "apt-get update/install" now and then stalls on
# a slow mirror (12 Oct 2026: Go (OpenCV) sat 45+ min in this step, normally
# about 1 min). apt has no overall deadline, so one stalled download blocked
# the job until it was cancelled. Here every attempt has a deadline, network
# reads time out, and a stalled attempt is retried.
#
# Usage: scripts/ci/apt-install.sh pkg1 pkg2 ...
set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "usage: $0 package..." >&2
  exit 2
fi

APT_OPTS=(
  -o Acquire::Retries=3
  -o Acquire::http::Timeout=30
  -o Acquire::https::Timeout=30
  -o DPkg::Lock::Timeout=120
)
ATTEMPTS="${APT_ATTEMPTS:-3}"
DEADLINE="${APT_DEADLINE:-480}" # seconds per attempt (update + install)

export DEBIAN_FRONTEND=noninteractive
for i in $(seq 1 "$ATTEMPTS"); do
  echo "apt-install: attempt $i/$ATTEMPTS: $*"
  if timeout --kill-after=30 "$DEADLINE" sudo -E apt-get "${APT_OPTS[@]}" update -qq &&
     timeout --kill-after=30 "$DEADLINE" sudo -E apt-get "${APT_OPTS[@]}" install -y -qq "$@"; then
    exit 0
  fi
  echo "apt-install: attempt $i failed or timed out" >&2
  # A killed apt can leave dpkg half-configured; finish that before retrying.
  sudo -E dpkg --configure -a || true
  sleep $((i * 10))
done
echo "apt-install: giving up after $ATTEMPTS attempts" >&2
exit 1
