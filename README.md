# NexaPulse

**English** | [简体中文](./README.zh-CN.md)

Self-hosted news pipeline: Telegram / RSS ingest, rule filters, optional LLM cleanup, and a public reading site.

- **Server**: Go
- **Database**: PostgreSQL
- **Admin**: React + TypeScript on host `127.0.0.1:8081` (never publish Admin to `0.0.0.0`)
- **Public**: React + TypeScript on `:8080`

## Quick start

```bash
cp .env.example .env
# Set real NEXA_ADMIN_TOKEN (≥24 chars) and POSTGRES_PASSWORD.
# Example placeholders are rejected at startup.

mkdir -p data/sessions
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

- Public site: `http://127.0.0.1:8080`
- Admin UI: `http://127.0.0.1:8081` with `NEXA_ADMIN_TOKEN`

Or pull a prebuilt image:

```bash
docker pull ghcr.io/kiteyuan/nexapulsebot:latest
```

Set `NEXA_IMAGE=ghcr.io/kiteyuan/nexapulsebot:latest` and run `docker compose up -d` (build overlay not required).

## Security notes

- Never commit `.env`, Telegram sessions under `data/sessions/`, or Postgres data.
- Inside Docker, Admin listens on `:8081` so port mapping works; compose publishes **only** `127.0.0.1:8081:8081`. Do not change that to `0.0.0.0:8081`.
- Put a reverse proxy with HTTPS in front of Public `:8080` for internet exposure.
- `NEXA_IMAGE_BASE_URL` defaults to a public image host example; set your own bed and keep `NEXA_IMAGE_AUTH` out of git.
- Telethon `.session` files cannot be reused. Log in again from the admin UI.

See [简体中文](./README.zh-CN.md) for layout and development notes.

## License

[MIT](./LICENSE)
