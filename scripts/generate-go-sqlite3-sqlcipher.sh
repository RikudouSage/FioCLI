#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
module_dir=$(go list -m -f '{{.Dir}}' github.com/mattn/go-sqlite3)
replacement_dir="$repo_root/build/go-modules/github.com/mattn/go-sqlite3"
patched_file="$replacement_dir/sqlite3.go"
modfile="$repo_root/build/go-sqlite3-sqlcipher.mod"
sumfile="$repo_root/build/go-sqlite3-sqlcipher.sum"

mkdir -p "$replacement_dir"
if [ -f "$patched_file" ]; then
	chmod u+w "$patched_file"
fi
cp -R "$module_dir/." "$replacement_dir/"
chmod -R u+w "$replacement_dir"

(
	cd "$replacement_dir"
	patch --quiet -p0 < "$repo_root/patches/go-sqlite3-sqlcipher-overlay.patch"
)

gofmt -w "$patched_file"

mkdir -p "$(dirname -- "$modfile")"
cp "$repo_root/go.mod" "$modfile"
cp "$repo_root/go.sum" "$sumfile"

{
	printf '\n'
	printf 'replace github.com/mattn/go-sqlite3 => %s\n' "$replacement_dir"
} >> "$modfile"

printf '%s\n' "$modfile"
