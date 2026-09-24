#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
out=$root/web/public
wasm=$out/gocalc.wasm

mkdir -p "$out"
GOOS=js GOARCH=wasm go build -o "$wasm" "$root/wasm"

loader=$(go env GOROOT)/lib/wasm/wasm_exec.js
if [ -f "$loader" ]; then
	cp "$loader" "$out/wasm_exec.js"
else
	echo "wasm_exec.js was not found at $loader; copy it beside gocalc.wasm by hand" >&2
fi

raw=$(wc -c < "$wasm" | tr -d ' ')
gzipped=$(gzip -9 -c "$wasm" | wc -c | tr -d ' ')
budget=4194304

mib() { awk -v b="$1" 'BEGIN { printf "%.2f MiB", b / 1048576 }'; }

echo "$wasm"
echo "  raw       $raw bytes ($(mib "$raw"))"
echo "  gzipped   $gzipped bytes ($(mib "$gzipped"))"
if [ "$gzipped" -le "$budget" ]; then
	echo "  budget    within $(mib "$budget") compressed"
else
	echo "  budget    OVER $(mib "$budget") compressed by $(mib $((gzipped - budget)))"
fi
