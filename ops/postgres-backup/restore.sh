#!/usr/bin/env sh
set -eu

require_env() {
  name="$1"
  eval "value=\${${name}:-}"
  if [ -z "${value}" ]; then
    echo "missing required environment variable: ${name}" >&2
    exit 2
  fi
}

restore_postgres() {
  if [ "$#" -ne 1 ]; then
    echo "usage: restore.sh /backups/<backup-file>.dump" >&2
    exit 2
  fi

  backup_file="$1"
  if [ ! -f "${backup_file}" ]; then
    echo "backup file not found: ${backup_file}" >&2
    exit 2
  fi

  require_env POSTGRES_HOST
  require_env POSTGRES_PORT
  require_env POSTGRES_DB
  require_env POSTGRES_USER
  require_env POSTGRES_PASSWORD

  echo "postgres restore started: db=${POSTGRES_DB} file=${backup_file}"
  PGPASSWORD="${POSTGRES_PASSWORD}" pg_restore \
    -h "${POSTGRES_HOST}" \
    -p "${POSTGRES_PORT}" \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" \
    --clean \
    --if-exists \
    --no-owner \
    "${backup_file}"
  echo "postgres restore finished: file=${backup_file}"
}

restore_postgres "$@"
