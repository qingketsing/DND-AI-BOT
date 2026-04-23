# Caddy Production Reverse Proxy

Production deployment uses Caddy as the public entry point. The backend app remains available only inside the Docker network on `app:8080`.

## Required Environment

```env
APP_DOMAIN=api.example.com
CORS_ALLOWED_ORIGINS=https://app.example.com
AUTH_COOKIE_SECURE=true
TRUSTED_PROXIES=172.16.0.0/12
METRICS_ALLOWED_CIDRS=172.16.0.0/12
```

## Start Production Stack

```bash
docker compose -f compose.yaml -f compose.prod.yaml up -d --build
```

## Expected Public Ports

Only Caddy should publish public ports:

```text
80/tcp
443/tcp
```

Do not publish `app:8080`, `postgres:5432`, or `redis:6379` in production.
