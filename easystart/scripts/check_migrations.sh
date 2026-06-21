#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${ROOT_DIR}/migrations"

if [[ ! -d "${MIGRATIONS_DIR}" ]]; then
  echo "migrations directory not found: ${MIGRATIONS_DIR}" >&2
  exit 1
fi

mapfile -t files < <(find "${MIGRATIONS_DIR}" -maxdepth 1 -type f -name '*.sql' -printf '%f\n' | sort)
if [[ ${#files[@]} -eq 0 ]]; then
  echo "no migration files found in ${MIGRATIONS_DIR}" >&2
  exit 1
fi

bad_files="$(
  printf '%s\n' "${files[@]}" \
    | grep -Ev '^[0-9]+_.+\.(up|down)\.sql$' || true
)"
if [[ -n "${bad_files}" ]]; then
  echo "migration filename check failed:" >&2
  printf '%s\n' "${bad_files}" | sed 's/^/  - /' >&2
  echo "expected format: 001_name.up.sql and 001_name.down.sql" >&2
  exit 1
fi

mapfile -t versions < <(printf '%s\n' "${files[@]}" | sed -E 's/^([0-9]+)_.*/\1/' | sort -u)

failed=0
for version in "${versions[@]}"; do
  up_count="$(printf '%s\n' "${files[@]}" | grep -Ec "^${version}_.+\.up\.sql$" || true)"
  down_count="$(printf '%s\n' "${files[@]}" | grep -Ec "^${version}_.+\.down\.sql$" || true)"
  if [[ "${up_count}" != "1" || "${down_count}" != "1" ]]; then
    if [[ "${failed}" == "0" ]]; then
      echo "migration numbering check failed:" >&2
    fi
    failed=1
    echo "  version ${version}: ${up_count} up, ${down_count} down; expected 1 up and 1 down" >&2
    printf '%s\n' "${files[@]}" | grep -E "^${version}_" | sed 's/^/    - /' >&2
  fi
done

if [[ "${failed}" != "0" ]]; then
  exit 1
fi

echo "migration numbering check: OK"
