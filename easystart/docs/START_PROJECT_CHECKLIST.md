# Start Project Checklist

Status: reusable template checklist

Use this after copying the template into a new project. Fill facts first, then implement.

## Profile Choice

- [ ] Choose one active profile before implementation.
- [ ] For a bot/backend project: keep `docs/PROJECT.md` as the active profile.
- [ ] For a web project: copy `docs/WEB_PROJECT.md` into `docs/PROJECT.md`, then update it with project facts.
- [ ] Remove or mark inactive profile files that should not guide the project.
- [ ] Confirm whether numbered working docs should stay local-only or be tracked for this project.

## Product

- [ ] Project name is recorded.
- [ ] Primary user-visible scenario is recorded.
- [ ] MVP boundary is recorded.
- [ ] Out-of-scope items are recorded.
- [ ] Owner acceptance criteria are recorded.

## Users And UX

- [ ] User roles are recorded.
- [ ] Admin/operator roles are recorded.
- [ ] Main screens/routes/commands are listed.
- [ ] Input flows are listed.
- [ ] Error and empty states are listed.
- [ ] Notification rules are listed.

## Data

- [ ] Main entities are listed.
- [ ] Ownership boundaries are recorded.
- [ ] Statuses and transitions are recorded.
- [ ] Limits and quotas are recorded.
- [ ] Required audit/log events are recorded.

## External Systems

- [ ] External APIs are listed.
- [ ] Source docs or local reference files are linked.
- [ ] Credentials/secrets are listed as env names, not values.
- [ ] Rate limits, retries, and timeout expectations are recorded.
- [ ] Webhook/domain/public callback needs are recorded.

## Owner Admin Platform

- [ ] Confirm whether this project will be managed from the shared owner admin platform.
- [ ] If yes, read `docs/ADMIN_PLATFORM_HANDSHAKE.md`.
- [ ] If yes, list MVP admin-visible entities and workflows.
- [ ] List deferred admin workflows separately from MVP.
- [ ] Record private admin contract version, transport, and network scope.
- [ ] Record first `health` and `capabilities` responses.
- [ ] Record first admin reads, commands, events, and long-running jobs.
- [ ] Record how the project validates admin-platform calls without storing secrets in docs.
- [ ] Confirm direct browser calls to project admin endpoints are out of scope.
- [ ] Confirm direct admin-platform writes into the project database for business actions are out of scope.

## Runtime And Deploy

- [ ] Service list is recorded.
- [ ] Docker Compose services are planned.
- [ ] `.env.example` variables are planned.
- [ ] Env/config examples use fenced `dotenv` or shell-style code blocks with visually distinct keys and values.
- [ ] Registry namespace is recorded: `DOCKERHUB_USER=ghcr.io/tuicli`.
- [ ] Local release flow is confirmed: `git add`, `git commit`, `git push`, `make push`.
- [ ] Server deploy flow is confirmed: `git pull --ff-only`, `make pull`.
- [ ] Staging/prod separation is recorded, when needed.
- [ ] Backup expectations are recorded, when needed.

## Checks

- [ ] Test command is recorded.
- [ ] Migration check command is recorded.
- [ ] Generation command is recorded, if generated files exist.
- [ ] Frontend/browser check is recorded, if this is a web project.
- [ ] Server smoke check is recorded, if production deploy is in scope.

## Documentation

- [ ] `docs/PROJECT.md` reflects the chosen profile and real project facts.
- [ ] First implementation/change doc exists when the change is large.
- [ ] Open questions are listed instead of guessed.
- [ ] Deferred items are listed separately from MVP.

## Checklist

- [x] Profile-choice rule included.
- [x] Product intake included.
- [x] UX/data/runtime intake included.
- [x] Owner admin-platform intake included.
- [x] Checks and documentation intake included.
