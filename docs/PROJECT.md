# Project Profile

Status: bot-first template

This file is tracked in git and defines the project-specific operating profile for Codex. Owner-facing communication stays in Russian; repository documentation is written in English unless the owner says otherwise.

Default use: Telegram/backend bot projects. `AGENTS.md` is shared across project types. For web projects, copy `docs/WEB_PROJECT.md` into this file before implementation, then remove or mark inactive profile files so only one project profile guides the work.

## Product Intake

Before implementation, fill these facts:
- profile choice from `docs/START_PROJECT_CHECKLIST.md`;
- product goal and primary user-visible scenario;
- bot roles, user roles, and access rules;
- main screens, commands, and expected user inputs;
- data entities and ownership boundaries;
- owner admin-platform integration needs: managed entities, private admin commands, audit/events, and rollout priority;
- external APIs and their source documentation;
- deployment target, domain/webhook needs, and server constraints;
- acceptance gates for the first usable version.

## Stack

- Language: Go.
- Database: PostgreSQL.
- Runtime: Docker Compose.
- Default services: backend, Telegram bot, optional admin bot, PostgreSQL.
- Default service style: backend services with explicit boundaries, migrations, tests, and observable runtime behavior.
- Default local release flow: `git add`, `git commit`, `git push`, `make push`.
- Default server update flow: `git pull --ff-only`, then `make pull`.
- Default registry namespace: `DOCKERHUB_USER=ghcr.io/tuicli`.
- Default image tag: current git short SHA, also publish `latest`.
- Server must not build source code during normal deploy; it pulls already built images.
- Add project-specific frameworks, APIs, queues, caches, and deployment details here after the project is created.

## Architecture

Default target: clean, modular, layered architecture with high scaling potential.

Preferred direction:
- domain/business rules do not depend on transport, storage, frameworks, or external APIs;
- handlers/controllers translate input and output only;
- application services coordinate use cases and transactions;
- repositories own persistence details;
- clients/adapters own external systems;
- background workers have explicit ownership, limits, retries, and observability;
- modules expose narrow contracts and avoid shared mutable state;
- projects that may be managed from the shared owner admin platform expose explicit private admin contracts;
- admin contracts describe capabilities first, then concrete commands, so unknown future admin scenarios do not block the MVP.

Avoid:
- business logic hidden inside HTTP/bot handlers;
- direct SQL or external API calls scattered across unrelated packages;
- direct owner admin-platform writes into project databases for business actions;
- direct browser calls to project admin endpoints;
- copied full admin panels per project when the shared owner admin platform is the intended operator UI;
- cyclic dependencies between modules;
- global state for request-specific behavior;
- abstractions created before there is a real second use case or real complexity to remove.

Default layout:
- `cmd/backend`: backend entry point, migrations, HTTP/gRPC server, background workers;
- `cmd/bot`: user Telegram bot entry point;
- `cmd/admin-bot`: admin Telegram bot entry point, when needed;
- `api/admin/v1`: private admin contract definitions, when the project is managed by the shared owner admin platform;
- `internal/domain`: business entities, policies, and pure rules;
- `internal/app`: use cases and application services;
- `internal/bot`: Telegram routing, UI, state, middleware, keyboards;
- `internal/storage/postgres`: PostgreSQL repositories and migration integration;
- `internal/config`: environment parsing and validation;
- `migrations`: SQL migrations;
- `configs`: runtime config files;
- `docs`: project profile and working docs.

## Owner Admin Platform Integration

Use this section only when the project should be managed from the shared owner admin platform. See `docs/ADMIN_PLATFORM_HANDSHAKE.md` for the reusable contract standard.

Target boundary:
- browser traffic goes to the shared admin platform, not directly to this project;
- the shared admin platform owns operator UI, GitHub/login policy, shared permissions, shared workflows, and admin audit;
- this project owns its business rules and exposes explicit private admin commands;
- this project validates admin-platform calls before executing any admin command;
- the admin platform must not write directly into this project's database for business actions.

Default first transport:
- same-server Docker Compose private network with internal HTTP/JSON;
- do not expose project admin endpoints as public browser APIs;
- keep transport replaceable so later deployments can move to private HTTPS, VPN/WireGuard/Tailscale, mTLS, or gRPC without changing the operator UI.

Minimum private admin contract:
- `GET /admin/v1/health`: service availability;
- `GET /admin/v1/capabilities`: supported admin resources, actions, and optional UI hints;
- resource reads needed by the MVP, such as users, orders, campaigns, or settings;
- explicit command endpoints for business actions, such as grant/revoke flags, block/unblock, resend, recalculate, or enqueue delivery;
- event/timeline reads for project events and user events;
- job create/status endpoints for long-running operations.

Contract rules:
- start with capabilities and the smallest real MVP commands;
- record deferred commands instead of guessing future buttons;
- every mutating command must have authorization, audit intent, input validation, timeout, and observable success/failure result;
- long-running commands return a job ID and expose status;
- never log secrets, tokens, raw private payloads, or admin auth material.

## Development Rules

- Start from the user-visible behavior or operational goal.
- Confirm current behavior in code, docs, logs, or owner-provided facts before changing it.
- Keep each change tied to a measurable result.
- Prefer small, reversible steps.
- Add tests around business rules, data boundaries, and regressions.
- Add logs/traces when a scenario cannot be diagnosed from existing signals.
- For new projects, write the first working docs before implementation.
- For large changes, write a decision/change doc first, then implement, then verify docs against code.
- Local runtime is not assumed. Prefer unit tests and Docker image builds locally; live Telegram/API checks happen on the server or staging when approved.
- Do not add GitHub MCP context unless the owner explicitly asks for PR, issue, review, or CI work.

## Deployment

Canonical Make targets to keep or adapt:
- `make test`: check migrations and run Go tests;
- `make check-migrations`: validate SQL migration numbering and up/down pairs;
- `make build`: build deployable Docker images;
- `make push`: require clean worktree, build images, push `<image>:<sha>` and `<image>:latest`;
- `make pull`: server-side `git pull --ff-only`, pull images by current short SHA, start services with `--no-build`;
- `make deploy-tag tag=<sha|latest>`: rollback or explicit deploy;
- `make logs service=<name>` and `make ps`: operational checks.

Deployment invariants:
- `.env` is outside git and contains secrets plus `DOCKERHUB_USER=ghcr.io/tuicli`;
- env/config examples in docs, chat, and diagnostic notes must be fenced as `dotenv` or shell-style code blocks so keys are visually distinct from values;
- compose uses PostgreSQL as a service and persistent named volumes;
- production/staging must have isolated `.env`, container names or compose project names, databases, volumes, bot tokens, and ports;
- migrations run automatically at backend start or through one documented command;
- `make pull` should verify that running containers use the expected image tag when practical.

## Bot Archetypes

Use this section for the default bot-first project.

Reference sources in this template workspace:
- `../bladogram`: main bot/runtime/deploy reference;
- `PINGO/docs/04-bot-ux-reference.md`: compact bot UX reference;
- `PINGO/internal/bot/ui`: reusable shape for UI manager/tracker;
- `PINGO/internal/bot/safehtml`: safe HTML escaping helpers.

Package direction:
- handlers translate Telegram updates into use cases and render screens;
- state store owns FSM state, TTL, and per-user/per-chat data;
- UI manager owns send/edit/upsert/delete behavior for the current UI message;
- keyboards package owns callback IDs and reusable button builders;
- middleware owns auth/access guard, callback ack tracking, logging, and update scope;
- app services own business scenarios;
- repositories own persistence.

Bot UX invariants:
- keep one current UI message per user/chat when practical;
- command or text input sends a new UI message and replaces the previous UI message;
- callback navigation edits the clicked message;
- if edit fails because Telegram cannot change message type, fallback to delete and send;
- ignore `message is not modified` as a user-visible error;
- always acknowledge callbacks once;
- text/file input must be routed through explicit FSM state;
- state must have TTL;
- input states must be marked so notifications or background pushes do not hijack the flow;
- `Back` uses explicit back target and optional entity/page id, not implicit stack magic;
- message deletion is best-effort and must not break the scenario;
- do not delete user messages by default; do it only for an explicitly documented scenario.

Bot copy and controls:
- screen title repeats the button emoji and name, uses bold formatting where Telegram HTML is used, then one blank line;
- major semantic blocks are separated by one blank line;
- dense cards use short label/value lines;
- user-facing errors are short, non-technical, and tell the next action;
- do not over-explain interface mechanics: the button label carries the action, and the message carries only the context needed to recognize the object and choose the action;
- do not mention standard workflow consequences such as notifications, status changes, or what happens after pressing a button, unless the consequence changes the user's decision;
- default action buttons prefer monochrome symbols over colored status emoji: `◀️ Назад`, `✖️ Отмена`, `👌 Понятно`, `✔️ Подтвердить`, `🔄 Обновить`;
- use `✔️`/`✖️` as the default action-button markers instead of `✅`/`❌`; colored status emoji remain acceptable inside message text when they represent state, not button styling;
- use semantic button styles when the Telegram client/API/library supports them: `success` / green for positive actions, `danger` / red for destructive or negative actions, `primary` / blue only for the main call to action;
- do not use transparent/unstyled buttons with text; transparent/unstyled buttons are only acceptable for pagination or other icon-only controls without text labels;
- omit button style only for pagination or other icon-only controls without text labels;
- pagination uses stable hollow arrow buttons such as `◁` and `▷` plus a no-op page indicator; vertical reordering uses `△` and `▽`;
- no-op buttons only acknowledge the callback and do not change the screen.

Copy example:
- Bad: `Second participant will receive a notification about early timer closing.`
- Good: show only the action, loan, date, and worker.

Runtime rules:
- background workers have explicit limits, retry budget, context cancellation, and logs;
- external API calls have timeouts, typed errors, and no secret logging;
- long operations immediately show a waiting state or callback toast;
- each observable scenario should show ingress, handler/service, internal calls, external calls, and result.

Examples to add:
- final menu map;
- final button labels;
- input validation rules;
- notification behavior;
- admin/user role model;
- production diagnostic commands.

Anti-examples to add:
- business logic in handlers;
- unbounded goroutines per update;
- hidden cross-chat global state;
- mixed UI text, persistence, and external calls in one function;
- retry loops without limits, context, or logs.

## Documentation

- `docs/PROJECT.md` is tracked and should stay compact.
- `docs/WEB_PROJECT.md` is the separate web-project profile; it is inactive unless the owner starts a web project or explicitly selects it.
- Numbered docs like `docs/01-topic.md` are Codex working docs and are ignored by git by default.
- Numbered docs should be English because they are mostly written by Codex for Codex.
- Every working doc must include `## Checklist`.
- Working docs should record target behavior, boundaries, risks, checks, open questions, and a checklist.
- If a fact is unknown, mark it as an open question instead of inventing a default.

## Checks

- Default: `make test` or `go test ./...`.
- Migrations: `make check-migrations`.
- Docker release: `make push`.
- Server deploy: `make pull`.
- Rollback: `make deploy-tag tag=<sha|latest>`.
- If protobuf or generated contracts change: run and document the project generation command.
- If local smoke is relevant: `make up`, `make logs`, `make down`.

## Examples

- Preferred env example format:
  ```dotenv
  POSTGRES_DB=admin_platform_db
  POSTGRES_USER=admin_platform_user
  POSTGRES_PASSWORD=change-me
  ```
- Bladogram-like deploy flow:
  - local: `git add`, `git commit`, `git push`, `make push`;
  - server: `git pull --ff-only`, `make pull`.
- Default `.env` registry line: `DOCKERHUB_USER=ghcr.io/tuicli`.
- Bot UI reference: one current interactive message, callback edit, input flow through FSM, explicit back targets.

## Anti-Examples

Add project-specific anti-examples here.

## Open Questions

- [ ] Fill in the actual service/module map after the project is initialized.
- [ ] Fill in exact Make targets after project files are generated.
- [ ] Fill in bot role/access model.
- [ ] Fill in menu map and user-visible copy.
- [ ] Fill in production/staging topology.
- [ ] Fill in owner admin-platform integration decision and private contract, when relevant.

## Checklist

- [x] Default stack recorded.
- [x] Architecture direction recorded.
- [x] Bot-first profile recorded.
- [x] Deploy flow recorded.
- [x] Bot UX invariants recorded.
- [x] Documentation policy recorded.
- [x] Owner admin-platform integration policy recorded.
