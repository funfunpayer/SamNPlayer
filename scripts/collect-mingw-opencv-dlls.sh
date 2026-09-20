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

# Direct + transitive MinGW deps of the binary.
mapfile -t libs < <(ldd "$exe" | awk '/\/mingw64\// {print $3}' | sort -u)
if [ "${#libs[@]}" -eq 0 ]; then
  echo "ldd found no /mingw64/ deps for $exe — is it a MinGW OpenCV build?" >&2
  exit 1
fi

for f in "${libs[@]}"; do
  [ -f "$f" ] || continue
  cp -n "$f" "$dest/" || cp "$f" "$dest/"
done

# Always include tracking/core siblings used at runtime by OpenCV plugins.
shopt -s nullglob
for f in /mingw64/bin/libopencv_*.dll; do
  cp -n "$f" "$dest/" 2>/dev/null || cp "$f" "$dest/"
done

echo "Collected $(ls -1 "$dest"/*.dll 2>/dev/null | wc -l) DLLs into $dest"
