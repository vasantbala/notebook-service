#!/usr/bin/env bash
set -euo pipefail

DB_URL="${DATABASE_URL:-postgres://notebook:notebook@localhost:5432/notebook?sslmode=disable}"

echo "Applying migrations to: $DB_URL"

for file in migrations/*.sql; do
  echo "  → $file"
  psql "$DB_URL" -f "$file"
done

echo "Migrations complete."
