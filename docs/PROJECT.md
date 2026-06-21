# Project Profile

Status: MVP implementation scaffolded locally

This file is tracked in git and defines the project-specific operating profile for Codex. Owner-facing communication stays in Russian; repository documentation is written in English unless the owner says otherwise.

Shorter is a standalone Telegram-admin link shortener. Admins paste many source URLs into the bot, review a parsed preview, confirm creation, and then manage/search/export short links. Public HTTP traffic resolves `BASE_URL/<code>` and redirects to the stored original URL.

## References

- `taxasya` is a read-only local reference for Telegram bot UX, PostgreSQL-backed short links, Docker Compose, Makefile deploy flow, and bot UI conventions.
- Use only UX/runtime patterns from the reference. Do not copy Pampadu-specific user, offer, public self-service, or tracking-link logic.
- Reference projects are read-only sources. Do not copy secrets from them or from chat into tracked files.

## Product

- Primary operator: Telegram admin from `ADMIN_USER_IDS`.
- Public user: anyone opening a short URL in a browser.
- Admin-visible result: short links are created from pasted URL batches and can be viewed, searched, exported, disabled, re-enabled, and deleted from Telegram.
- Public result: active short links redirect with HTTP `302`; disabled, deleted, or unknown codes return `404`.
- Out of scope for MVP: web admin panel, Telegram public-user mode, Taxasya integration, roles beyond admin whitelist, custom code editing, analytics dashboards, hard delete from PostgreSQL.

## Stack

- Language: Go.
- Database: PostgreSQL.
- Telegram Bot API library: `gopkg.in/telebot.v3`, following the Taxasya reference unless implementation finds a stronger reason to change.
- Runtime: Docker Compose on one server.
- Services: `bot`, `backend`, `migrate`, `postgres`.
- Service style: small modular monolith split into bot, application, domain, storage, config, and HTTP redirect packages.
- Config: environment variables for secrets and runtime values.
- Short-link shape: `BASE_URL/<code>`.
- Code generation: random URL-safe alphanumeric code, initially 8 characters, unique in PostgreSQL.

## Architecture

Default target: clean, modular, layered architecture without unnecessary framework weight.

Expected module map:
- `cmd/bot`: Telegram bot entry point.
- `cmd/backend`: HTTP redirect backend.
- `cmd/migrate`: migration runner, if not handled inside service startup.
- `internal/bot`: Telegram handlers, keyboards, callback routing, UI manager, and FSM/listener states.
- `internal/app`: use cases for parsing input, previewing batches, creating short links, listing/searching links, disabling/enabling/deleting links, and CSV export.
- `internal/domain`: short links, parsed input rows, validation rules, status values, and code generation policy.
- `internal/storage/postgres`: SQL repositories and migrations.
- `internal/config`: environment parsing and validation.
- `migrations`: SQL migrations.
- `scripts`: local validation and utility scripts.
- `docs`: project profile and working docs.

Avoid:
- business logic hidden inside HTTP/bot handlers;
- direct SQL scattered through bot or HTTP packages;
- public browser admin endpoints;
- shared mutable bot state without explicit TTL;
- Taxasya/Pampadu-specific entities in this project.

## Data Model

Planned tables:
- `short_links`: one row per unique original URL.
- `link_events`: lightweight audit/events for creation, duplicate preview, enable, disable, delete, and export actions.
- `short_link_clicks`: deferred unless click tracking is enabled during implementation.

`short_links` fields:
- `id BIGSERIAL PRIMARY KEY`
- `code TEXT NOT NULL UNIQUE`
- `original_url TEXT NOT NULL UNIQUE`
- `title TEXT NOT NULL`
- `status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'deleted'))`
- `created_by_telegram_id BIGINT NOT NULL`
- `updated_by_telegram_id BIGINT`
- `disabled_at TIMESTAMPTZ`
- `disabled_by_telegram_id BIGINT`
- `deleted_at TIMESTAMPTZ`
- `deleted_by_telegram_id BIGINT`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

Input parsing:
- Extract only the first HTTP/HTTPS URL from each input line.
- Ignore a leading list number such as `21.`.
- Text after the extracted URL becomes the title after trimming separators and whitespace.
- If no title remains, derive a short title from the URL host/path.
- Lines without URLs are reported in preview and are not written.

Duplicate policy:
- Duplicate `original_url` is not inserted again, regardless of existing link status.
- Preview must show duplicates separately, including the existing short URL and status, and explain that the row will be skipped.
- Duplicate `code` during random generation is an internal retry; after retry budget is exhausted, return a short admin-facing error.
- Existing code and original URL are not reused or regenerated in MVP, even after soft deletion.

## Bot UX

Access:
- Admin whitelist comes from `ADMIN_USER_IDS`.
- Non-admin users receive no response by default.
- Private chat is the expected admin surface.

Main menu:
- `➕ Добавить ссылки`
- `🔎 Найти`
- `🕘 Последние`
- `⬇️ CSV`

Add links flow:
1. Admin presses `➕ Добавить ссылки`.
2. Bot asks for a text block with one or more URLs.
3. Bot parses input and shows a preview before any write.
4. Preview groups valid new links, duplicate existing links, and invalid lines.
5. Admin presses `✔️ Подтвердить` or `✖️ Отмена`.
6. Bot creates only new links and returns a result summary plus the created short URLs.

List/search flow:
- Latest links are ordered newest first.
- Search uses one input field and matches code, title, or original URL.
- Results are paginated with 10 links per page.
- Default lists and search show `active` and `disabled` links. `deleted` links are hidden from normal lists unless a later explicit filter is added.
- Link buttons use this label shape: `<short title> - <short_url>`.
- Button title part is capped for compact Telegram display. If the title is longer than 7 visible characters, show the first 6 characters plus `..`.
- Link detail shows title, code, short URL, original URL, status, created time, and admin actions.
- Active link detail supports `⏸ Выключить` and `✖️ Удалить`.
- Disabled link detail supports `▶️ Включить` and `✖️ Удалить`.
- Delete is a soft delete: set `status='deleted'`, record deletion metadata, hide from default bot lists, and make the public short URL return `404`.

Bot UI conventions:
- Keep one current UI message per admin/chat when practical.
- Callback navigation edits the clicked message.
- Text input flows use explicit FSM/listener state with TTL and support cancel.
- Confirmation screens show exactly what will be written before writing.
- Always acknowledge callbacks once.
- Ignore `message is not modified` as a user-visible error.
- Telegram messages use HTML escaping and no link previews by default.
- Use `◀️ Назад`, `✖️ Отмена`, `✔️ Подтвердить`, `🔄 Обновить`.
- Use hollow arrows `◁` and `▷` plus a no-op page indicator for pagination.

## HTTP Redirect

- `GET /healthz` returns service health.
- `GET /<code>` resolves active short links and redirects with `302`.
- Unknown, invalid, disabled, or deleted codes return `404`.
- Codes should be constrained to URL-safe alphanumeric characters.
- Redirect handler must not expose internal errors or database details.
- Request logs should include code, result, HTTP status, elapsed time, and a request/correlation id when available.

## Export And Dumps

Telegram CSV export:
- Admin can request CSV from the bot.
- CSV includes at least `code`, `short_url`, `original_url`, `title`, `status`, `created_at`.
- CSV export must stream or cap safely if the table grows; exact limit is implementation-defined and recorded before release.

PostgreSQL dumps:
- Runtime must support standard PostgreSQL access through Docker Compose and env credentials.
- Backup/restore scripts are expected from the owner and should rely on documented PostgreSQL env names, not hard-coded credentials.
- Do not print database passwords in docs, logs, or diagnostic `comm`.

Useful dump queries:

```sql
SELECT code, original_url, title, status, created_at
FROM short_links
ORDER BY id;
```

```sql
SELECT code, original_url, title, status, created_at
FROM short_links
WHERE status = 'active'
ORDER BY id;
```

## Runtime Env

Required planned env:

```dotenv
TELEGRAM_BOT_TOKEN=change-me
ADMIN_USER_IDS=123456789,987654321
BASE_URL=https://short.example.com
DATABASE_URL=postgres://shorter:change-me@postgres:5432/shorter?sslmode=disable
POSTGRES_DB=shorter
POSTGRES_USER=shorter
POSTGRES_PASSWORD=change-me
BACKEND_HTTP_ADDR=:8080
BOT_POLL_TIMEOUT_SECONDS=30
MIGRATIONS_DIR=/app/migrations
DOCKERHUB_USER=ghcr.io/tuicli
```

Optional planned env:

```dotenv
CODE_LENGTH=8
ADD_LINKS_MAX_LINES=200
LINKS_PAGE_SIZE=10
CSV_EXPORT_MAX_ROWS=10000
```

## Development Rules

- Start from the user-visible Telegram behavior or public redirect behavior.
- Confirm current behavior in code, docs, logs, or owner-provided facts before changing it.
- Keep each change tied to a measurable result.
- Prefer small, reversible steps.
- Add tests around parsing, duplicate detection, code generation collision handling, search, CSV export, and redirect resolution.
- Add logs/traces when a scenario cannot be diagnosed from existing signals.
- Local runtime is not assumed. Prefer unit tests and Docker image builds locally; live Telegram and public-domain checks happen on the server after deploy unless the owner explicitly asks for a local probe.
- Do not commit `.env`, Telegram tokens, database passwords, dumps, logs, or raw production data.

## Deployment

Canonical Make targets to implement:
- `make test`: check migrations and run Go tests.
- `make check-migrations`: validate SQL migration numbering and up/down pairs.
- `make migrate`: apply migrations through Docker Compose.
- `make build`: build deployable Docker images.
- `make push`: require clean worktree, build images, push `<image>:<sha>` and `<image>:latest`.
- `make pull`: server-side deploy from the current checkout: pull images by current short SHA, run migrations, start services with `--no-build`. It does not run `git pull`.
- `make logs service=<name>` and `make ps`: operational checks.

Deployment invariants:
- `.env` is outside git and contains secrets plus `DOCKERHUB_USER=ghcr.io/tuicli`.
- First server install flow: owner runs `git clone`, fills `.env`, then runs `make pull`.
- Later server update flow: owner runs `git pull --ff-only`, then `make pull`.
- Compose uses PostgreSQL as a service and persistent named volumes.
- Production/staging must have isolated `.env`, compose project names, databases, volumes, bot tokens, and ports.
- Server must not build source code during normal deploy; it pulls already built images.
- Public HTTPS termination can be handled by nginx or another server proxy in front of `backend`.

## Owner Admin Platform

- Not part of MVP.
- This project is operated through the Telegram admin bot.
- If owner admin platform integration is requested later, add a private admin contract before exposing browser admin actions.

## Checks

- Default before commit after code exists: `make test`.
- Fallback: `go test ./...`.
- Migrations: `make check-migrations`.
- Docker release: `make push`.
- Server deploy: `make pull`.
- Live checks after deploy: Telegram admin add/search/disable/enable/delete/export flow and public redirect by one created short URL.

## Open Questions

- [ ] Confirm final production domain value for `BASE_URL`.
- [ ] Confirm final backup script shape after the owner provides the dump reference.
- [ ] Confirm whether click tracking is needed in MVP; current target keeps it deferred.

## Checklist

- [x] Active profile selected as Telegram/backend bot-first.
- [x] Product purpose recorded.
- [x] Taxasya reference boundary recorded.
- [x] Stack recorded.
- [x] Architecture direction recorded.
- [x] Main bot UX recorded.
- [x] Parsing and duplicate policy recorded.
- [x] Data model recorded.
- [x] Link status lifecycle recorded.
- [x] Public redirect behavior recorded.
- [x] Export and dump expectations recorded.
- [x] Runtime env names recorded.
- [x] Deployment target recorded.
- [x] Owner admin platform decision recorded.
- [x] MVP code scaffold implemented.
- [x] Local `make test` passed.
- [x] Local `docker compose config` passed without checked-in secrets.
