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

backup_postgres() {
  require_env POSTGRES_HOST
  require_env POSTGRES_PORT
  require_env POSTGRES_DB
  require_env POSTGRES_USER
  require_env POSTGRES_PASSWORD

  backup_dir="${POSTGRES_BACKUP_DIR:-/backups}"
  backup_prefix="${POSTGRES_BACKUP_PREFIX:-dnd_ai_bot}"
  retention_days="${POSTGRES_BACKUP_RETENTION_DAYS:-7}"

  mkdir -p "${backup_dir}"

  timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
  backup_file="${backup_dir}/${backup_prefix}_${POSTGRES_DB}_${timestamp}.dump"

  echo "postgres backup started: db=${POSTGRES_DB} file=${backup_file}"
  PGPASSWORD="${POSTGRES_PASSWORD}" pg_dump \
    -h "${POSTGRES_HOST}" \
    -p "${POSTGRES_PORT}" \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" \
    -Fc \
    -f "${backup_file}"

  if [ ! -s "${backup_file}" ]; then
    echo "postgres backup failed: empty backup file ${backup_file}" >&2
    exit 1
  fi

  find "${backup_dir}" \
    -name "${backup_prefix}_${POSTGRES_DB}_*.dump" \
    -type f \
    -mtime "+${retention_days}" \
    -exec rm -f {} \;

  echo "postgres backup finished: file=${backup_file}"
}

backup_postgres "$@"
