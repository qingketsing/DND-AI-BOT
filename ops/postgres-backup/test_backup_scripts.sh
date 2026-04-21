#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

FAKE_BIN="${TMP_DIR}/bin"
BACKUP_DIR="${TMP_DIR}/backups"
mkdir -p "${FAKE_BIN}" "${BACKUP_DIR}"

cat > "${FAKE_BIN}/pg_dump" <<'SCRIPT'
#!/usr/bin/env sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-f" ]; then
    shift
    out="$1"
  fi
  shift || true
done
if [ -z "${out}" ]; then
  echo "missing -f" >&2
  exit 2
fi
printf '%s\n' "$*" > "${PG_DUMP_ARGS_FILE}"
printf '%s\n' "${PGPASSWORD:-}" > "${PG_DUMP_PASSWORD_FILE}"
printf 'fake dump\n' > "${out}"
SCRIPT
chmod +x "${FAKE_BIN}/pg_dump"

cat > "${FAKE_BIN}/pg_restore" <<'SCRIPT'
#!/usr/bin/env sh
set -eu
printf '%s\n' "$*" > "${PG_RESTORE_ARGS_FILE}"
printf '%s\n' "${PGPASSWORD:-}" > "${PG_RESTORE_PASSWORD_FILE}"
SCRIPT
chmod +x "${FAKE_BIN}/pg_restore"

export PATH="${FAKE_BIN}:${PATH}"
export POSTGRES_HOST="postgres"
export POSTGRES_PORT="5432"
export POSTGRES_DB="dndbot"
export POSTGRES_USER="dnd"
export POSTGRES_PASSWORD="dndpass"
export POSTGRES_BACKUP_DIR="${BACKUP_DIR}"
export POSTGRES_BACKUP_PREFIX="dnd_ai_bot"
export POSTGRES_BACKUP_RETENTION_DAYS="7"
export PG_DUMP_ARGS_FILE="${TMP_DIR}/pg_dump_args"
export PG_DUMP_PASSWORD_FILE="${TMP_DIR}/pg_dump_password"
export PG_RESTORE_ARGS_FILE="${TMP_DIR}/pg_restore_args"
export PG_RESTORE_PASSWORD_FILE="${TMP_DIR}/pg_restore_password"

old_backup="${BACKUP_DIR}/dnd_ai_bot_dndbot_20000101T000000Z.dump"
printf 'old\n' > "${old_backup}"
touch -d '20 days ago' "${old_backup}" 2>/dev/null || touch -t 200001010000 "${old_backup}"

"${ROOT_DIR}/ops/postgres-backup/backup.sh"

new_count="$(find "${BACKUP_DIR}" -name 'dnd_ai_bot_dndbot_*.dump' -type f | wc -l | tr -d ' ')"
if [ "${new_count}" != "1" ]; then
  echo "expected exactly one retained backup, got ${new_count}" >&2
  exit 1
fi
if [ -e "${old_backup}" ]; then
  echo "expected old backup to be deleted" >&2
  exit 1
fi
if [ "$(cat "${PG_DUMP_PASSWORD_FILE}")" != "dndpass" ]; then
  echo "expected backup script to pass PGPASSWORD" >&2
  exit 1
fi

backup_file="$(find "${BACKUP_DIR}" -name 'dnd_ai_bot_dndbot_*.dump' -type f | head -n 1)"
"${ROOT_DIR}/ops/postgres-backup/restore.sh" "${backup_file}"

restore_args="$(cat "${PG_RESTORE_ARGS_FILE}")"
case "${restore_args}" in
  *"--clean"*"--if-exists"*"--no-owner"* ) ;;
  *)
    echo "expected restore args to include --clean --if-exists --no-owner, got ${restore_args}" >&2
    exit 1
    ;;
esac
if [ "$(cat "${PG_RESTORE_PASSWORD_FILE}")" != "dndpass" ]; then
  echo "expected restore script to pass PGPASSWORD" >&2
  exit 1
fi

echo "backup script tests passed"
