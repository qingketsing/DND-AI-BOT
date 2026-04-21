# PostgreSQL Backup

This container runs daily PostgreSQL logical backups with `pg_dump -Fc`.

## Schedule

The cron schedule is fixed at UTC 03:00:

```cron
0 3 * * *
```

## Manual Backup

```bash
docker compose run --rm postgres-backup /ops/postgres-backup/backup.sh
```

Backups are written to:

```text
./backups/postgres
```

## Manual Restore

Restoring is destructive. Stop the API before restoring if this is a live environment.

```bash
docker compose stop app
docker compose run --rm postgres-backup /ops/postgres-backup/restore.sh /backups/<backup-file>.dump
docker compose start app
```

## Verify a Backup

```bash
docker compose run --rm postgres-backup pg_restore --list /backups/<backup-file>.dump
```

## Retention

`POSTGRES_BACKUP_RETENTION_DAYS` controls local retention. The default compose value keeps 7 days.
