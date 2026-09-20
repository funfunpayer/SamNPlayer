#!/usr/bin/env bash
# Collect MinGW DLLs needed to run a CGO OpenCV binary beside the .exe.
# Usage: collect-mingw-opencv-dlls.sh <exe> <dest-dir>
set -euo pipefail

exe=${1:?exe path}
dest=${2:?dest dir}
mkdir -p "$dest"

if ! command -v ldd >/dev/null 2>&1; then
  echo "ldd not found (need MSYS2/MinGW shell)" >&2
  exit 1
fi

count=0
while read -r f; do
  [ -n "$f" ] || continue
  [ -f "$f" ] || continue
  cp -f "$f" "$dest/"
  count=$((count + 1))
done < <(ldd "$exe" | awk '/\/mingw64\// {print $3}' | sort -u)

# Always include all OpenCV module DLLs (tracking loads siblings at runtime).
shopt -s nullglob
for f in /mingw64/bin/libopencv_*.dll; do
  cp -f "$f" "$dest/"
  count=$((count + 1))
done

n=$(ls -1 "$dest"/*.dll 2>/dev/null | wc -l | tr -d ' ')
echo "Collected ${n} DLLs into $dest (ldd hits≈${count})"
if [ "$n" -lt 3 ]; then
  echo "too few DLLs — OpenCV link likely missing" >&2
  exit 1
fi
