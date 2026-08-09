#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source_dir="$root/third_party/ooz"
output_dir="$root/internal/sav/assets"
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

mkdir -p "$output_dir"
cp "$source_dir"/*.cpp "$source_dir"/*.h "$work_dir"/
sed -i '/SDKDDKVer.h/d' "$work_dir/targetver.h"
sed -i 's|<Windows.h>|<windows.h>|' "$work_dir/stdafx.h"
sed -i '/#include <stdio.h>/a #include <sys/stat.h>' "$work_dir/stdafx.h"

x86_64-w64-mingw32-g++ -O2 -std=c++17 -mwindows -static-libgcc -static-libstdc++ \
  "$work_dir/stdafx.cpp" "$work_dir/kraken.cpp" "$work_dir/lzna.cpp" "$work_dir/bitknit.cpp" \
  -o "$output_dir/ooz_windows_amd64.exe"
