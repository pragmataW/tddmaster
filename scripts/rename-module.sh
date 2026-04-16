#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 2 ]]; then
	echo "Usage: $0 <old-module-path> <new-module-path>" >&2
	exit 1
fi

old_module_path=$1
new_module_path=$2

if [[ "$old_module_path" == "$new_module_path" ]]; then
	echo "Module paths are identical; nothing to do."
	exit 0
fi

go mod edit -module "$new_module_path"

while IFS= read -r file; do
	perl -0pi -e "s#\\Q$old_module_path\\E#$new_module_path#g" "$file"
done < <(
	rg -l --hidden \
		--glob '!.git/**' \
		--glob '!.venv/**' \
		--glob '!.code-review-graph/**' \
		--glob '!.codex/**' \
		--glob '!.claude/**' \
		"$old_module_path" .
)

go mod tidy
