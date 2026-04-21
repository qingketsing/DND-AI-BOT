FROM postgres:16-alpine

RUN apk add --no-cache dcron tzdata

WORKDIR /ops/postgres-backup

COPY ops/postgres-backup/backup.sh /ops/postgres-backup/backup.sh
COPY ops/postgres-backup/restore.sh /ops/postgres-backup/restore.sh
COPY ops/postgres-backup/crontab /etc/crontabs/root

RUN chmod +x /ops/postgres-backup/backup.sh /ops/postgres-backup/restore.sh

CMD ["crond", "-f", "-l", "8"]
