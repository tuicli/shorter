# Web Project Profile

Status: separate web template profile

This file is a separate starting profile for web projects. It is inactive for bot-first projects unless the owner explicitly selects it. When a new web project starts, copy this file into that project's `docs/PROJECT.md`, then remove or mark inactive profile files so only one project profile guides implementation.

## Product Intake

Before implementation, fill these facts:
- product type: landing, admin panel, SaaS tool, content site, marketplace, or mixed app;
- primary visitor/user scenario and conversion goal;
- first viewport content and required visual references;
- admin/content-management needs;
- owner admin-platform integration needs, when this web project should be managed from the shared owner admin platform;
- legal, analytics, SEO, and cookie requirements;
- public domain, routes, and nginx/certbot constraints;
- acceptance gates for desktop and mobile.

## Stack

Default web stack:
- Backend: Go.
- Database: PostgreSQL.
- Frontend: Next.js, TypeScript, Tailwind CSS.
- Runtime: Docker Compose.
- Edge/proxy: nginx.
- Optional admin UI: custom components first; shadcn/ui may be used when it reduces real admin complexity.

Default release flow:
- local: `git add`, `git commit`, `git push`, `make push`;
- server: `git pull --ff-only`, `make pull`;
- registry namespace: `DOCKERHUB_USER=ghcr.io/tuicli`;
- images are tagged with current git short SHA and `latest`;
- server deploy pulls images and starts with `--no-build`.

## Architecture

Default runtime direction:
- nginx routes public traffic to the frontend container and API traffic to the Go backend;
- frontend owns public pages and admin UI;
- backend owns admin API, domain logic, redirects/tracking, and migrations;
- PostgreSQL stores content, admin users/sessions, analytics events, and settings;
- public routes and admin routes are separated clearly.

Preferred layout:
- `cmd/api`: backend HTTP API entry point;
- `cmd/migrate`: migration runner, if migrations are not run at API start;
- `cmd/admin-bootstrap`: first admin creation, when needed;
- `internal/domain`: business rules;
- `internal/app`: use cases;
- `internal/httpserver`: HTTP handlers and middleware;
- `internal/storage`: repositories;
- `frontend/app`: Next.js routes;
- `frontend/lib`: frontend data/contracts/helpers;
- `migrations`: SQL migrations;
- `nginx`: container nginx config;
- `deploy`: system nginx examples and server notes.

Avoid:
- business logic inside React components or HTTP handlers;
- frontend-only source of truth for data that backend must validate;
- direct browser calls to private project admin endpoints;
- direct shared-admin writes into this project's database for business actions;
- public pages that depend on admin-only state without clear fallbacks;
- storing secrets in frontend env variables exposed to the browser;
- decorative layout that blocks scanning, conversion, or repeated admin work.

## Owner Admin Platform Integration

Use this section only when this web project should be managed from the shared owner admin platform. See `docs/ADMIN_PLATFORM_HANDSHAKE.md` for the reusable contract standard.

Target boundary:
- public visitors use this project's public UI;
- owner/operator traffic goes to the shared admin platform, not directly to private project admin endpoints;
- the shared admin platform owns operator UI, login policy, shared permissions, shared workflows, and admin audit;
- this project owns its business rules and exposes explicit private admin commands;
- this project validates admin-platform calls before executing any admin command;
- the shared admin platform must not write directly into this project's database for business actions.

Default first transport:
- same-server Docker Compose private network with internal HTTP/JSON;
- keep transport replaceable so later deployments can move to private HTTPS, VPN/WireGuard/Tailscale, mTLS, or gRPC without changing the operator UI.

Minimum private admin contract:
- `GET /admin/v1/health`;
- `GET /admin/v1/capabilities`;
- resource reads needed by the MVP;
- explicit command endpoints for business actions;
- event/timeline reads;
- job create/status endpoints for long-running operations.

## Web UX And Design

Start from the user-visible result:
- what the visitor sees in the first viewport;
- what action the visitor should take;
- what admin/operator changes and where it appears publicly;
- what remains out of scope for this version.

Landing/page rules:
- first screen must show the actual product, offer, place, person, or concrete visual subject;
- avoid generic gradient/SVG-only heroes when a real/generated image or actual product media is needed;
- hero headline should be the brand/product/category/offer, with value props in supporting copy;
- leave a hint of the next section visible on mobile and desktop;
- do not build a marketing shell when the user asked for a usable app/tool.

Operational/admin UI rules:
- admin interfaces should be dense, calm, predictable, and scan-friendly;
- use tables, filters, tabs, menus, toggles, and forms where they fit the workflow;
- avoid oversized hero sections, decorative cards, and marketing composition inside admin tools;
- keep repeated items as cards only when cards improve comparison or scanning;
- do not put cards inside cards.

Responsive checks:
- text must fit inside buttons/cards on mobile and desktop;
- fixed-format elements need stable dimensions through grid tracks, aspect ratio, min/max, or container constraints;
- no overlapping text, controls, sticky headers, modals, or floating actions;
- do not scale font size with viewport width;
- letter spacing should stay `0` unless a specific brand rule says otherwise.

Visual assets:
- websites must use relevant bitmap assets or real media when the subject should be inspectable;
- generated images are acceptable when no real assets exist and the owner approves the direction;
- public offer/product images should live in a documented public assets directory;
- admin-selected images should use existing project assets by default; upload can be added later when required.

## Development Rules

- Use current official docs before framework-specific implementation decisions or fixes.
- Use Context7 for Next.js, Tailwind CSS, shadcn/ui, and other active libraries when syntax or setup matters.
- Keep frontend and backend contracts explicit.
- Add backend tests for business rules and data boundaries.
- Use Docker build as the frontend production check when that is the project convention.
- Use Playwright for visual/reference analysis, screenshots, responsive checks, and bug reproduction.
- The owner remains the primary visual reviewer.

## Deployment

Canonical Make targets to keep or adapt:
- `make up`: local Docker stack when local browser review is needed;
- `make down`: stop local stack;
- `make test`: backend and frontend checks;
- `make build`: build deployable images;
- `make push`: require clean worktree, build images, push `<image>:<sha>` and `<image>:latest`;
- `make pull`: server-side `git pull --ff-only`, pull images by current short SHA, start with `--no-build`;
- `make deploy-tag tag=<sha|latest>`: rollback or explicit deploy;
- `make migrate`: run migrations if not automatic;
- `make logs service=<name>` and `make ps`: operational checks.

Server defaults:
- system nginx proxies the public domain to local Docker nginx/API ports;
- internal admin panels and observability bind to `127.0.0.1` unless explicitly exposed;
- certbot can manage public TLS after DNS points to the server;
- staging and production use isolated `.env`, compose project names, databases, volumes, host ports, and domains.

## Documentation

Required docs for non-trivial web projects:
- product brief with scenarios and conversion goal;
- design direction/reference analysis;
- architecture and data model;
- admin/content model when admin exists;
- implementation plan with gates;
- deployment notes after real server facts are known.

Each working doc must include `## Checklist`.

## Checks

Default checks:
- backend: `go test ./...` or `make test-go`;
- frontend: production Docker build or documented frontend check;
- browser: Playwright screenshot/responsive review for public UI changes;
- migrations: documented migration command;
- deploy: `make push` locally, `make pull` on server.

## Open Questions

- [ ] Choose actual public domain and route map.
- [ ] Choose whether local Docker review is required before server deploy.
- [ ] Choose admin auth model.
- [ ] Choose whether this project is managed from the shared owner admin platform.
- [ ] Choose analytics and legal/cookie requirements.
- [ ] Fill exact Make targets after project generation.

## Checklist

- [x] Separate web profile recorded.
- [x] Default web stack recorded.
- [x] Web deploy flow recorded.
- [x] Web UX rules recorded.
- [x] Browser/Playwright check policy recorded.
- [x] Owner admin-platform integration policy recorded.
