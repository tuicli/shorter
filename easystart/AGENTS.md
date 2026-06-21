# AGENTS

Last updated: 2026-06-04

Purpose: compact, reusable operating rules for Codex in this repository template.

## 0) Language And Brevity

- Talk with the owner in Russian.
- Keep owner-facing answers short by default: answer the current question, list only needed facts, then the next action.
- Write repository docs in English unless the owner explicitly requests otherwise.
- Keep docs concise: decisions, facts, checks, and open questions; no narrative filler.

## 1) Authority Order

Use this priority order:
1. Explicit owner decision in the current dialog.
2. This `AGENTS.md`.
3. `docs/PROJECT.md`.
4. Current numbered working docs in `docs/`, when they exist.
5. Code as the source of current behavior.

If docs and code disagree, record the mismatch, align target behavior with the owner, then update docs and code consistently.

## 2) Project Profile

- Always read `docs/PROJECT.md` before non-trivial planning or implementation.
- Before any bot UI, button, screen, message, or copy change, reread `docs/PROJECT.md`, especially bot UX/copy rules, and include that check in `Чек перед правками`.
- `docs/PROJECT.md` is the tracked project profile and the default active profile.
- `AGENTS.md` is shared across bot and web projects.
- Default project type is Telegram/backend bot-first. For a bot project, keep bot rules in `docs/PROJECT.md`.
- Web projects use `docs/WEB_PROJECT.md` as a separate source profile. For a web project, copy `docs/WEB_PROJECT.md` into the active `docs/PROJECT.md` before implementation.
- After the active profile is chosen, remove or mark inactive profile files so they do not guide implementation.
- Numbered docs in `docs/` are working memory for Codex. They should be English, compact, and ordered like `docs/01-topic.md`, `docs/02-topic.md`.
- Keep numbered docs out of git by default unless the owner explicitly decides otherwise.
- When starting a new project, fill `docs/START_PROJECT_CHECKLIST.md` and then `docs/PROJECT.md` first.
- For projects that may be managed from the shared owner admin platform, read `docs/ADMIN_PLATFORM_HANDSHAKE.md` and fill the owner admin integration section in `docs/PROJECT.md` before implementation.
- If stack details, commands, architecture, or runtime facts are unknown, record them as open questions instead of inventing defaults.
- Do not keep two competing active project profiles. One profile must be named as the current source of truth before implementation starts.

## 3) Engineering Defaults

- Default stack: Go and PostgreSQL.
- Default runtime/deploy shape: Docker Compose, one compose file, server deploy from pushed images, no source build on the server.
- Default registry namespace: `DOCKERHUB_USER=ghcr.io/tuicli`.
- Default local release flow after checks: `git add`, `git commit`, `git push`, `make push`.
- Default server update flow: `git pull --ff-only`, `make pull`.
- Prefer clean architecture with modular, layered boundaries and explicit dependencies.
- Keep business rules independent from transport, storage, framework, and external API details.
- Design for future scale, but do not add abstractions that do not remove real complexity now.
- Use existing project patterns before introducing new ones.
- Do not invent libraries, modules, causes, or sources before confirming them in code, docs, logs, or from the owner.
- Do not bake project/runtime facts into business code; keep environment-specific values in config, schema facts in migrations, stable protocol/status values in named constants, and project decisions in `docs/PROJECT.md`.
- Default owner admin integration: the shared admin platform owns browser UI, auth, permissions, and shared audit; each managed project exposes a private admin contract, validates admin-platform calls, and executes its own business actions. Do not use direct browser-to-project admin calls or direct admin-platform writes into project databases for business actions.

## 3.1 GitHub And External Tools

- Do not use GitHub MCP/app tools by default.
- Local git inspection is enough by default: `git status`, `git diff`, `git log`.
- Use GitHub MCP/app tools only when the owner explicitly asks to inspect PRs, issues, review comments, CI checks, repository metadata, or to create/update a PR.
- Do not spend context on GitHub metadata when the owner handles GitHub manually.

## 4) Hard Stop Before Edits

- Until the owner explicitly writes `подтверждаю`, work in analysis mode only.
- Before `подтверждаю`, do not edit files, DB, environment, migrations, generated files, or run commands that write state.
- Exception: overwriting `comm` for server diagnostics is allowed under section 7.
- Before edits, provide `Чек перед правками`: goal, boundaries, DoD, files, checks.
- If this hard stop is violated, revert only your own changes from the current task, then return to analysis and wait for `подтверждаю`.

## 5) Interaction Rules

- For new tasks, start with `Как понял задачу`: what changes, where the user sees it, and what is out of scope.
- If the request is ambiguous, ask one short clarifying question.
- For bug or incident diagnosis: facts first, conclusion second. Separate confirmed facts from hypotheses.
- Never present a hypothesis as a confirmed root cause.
- If facts are insufficient, state exactly what data is missing and request the smallest needed log, trace, or scenario.
- For incident answers, use this order: direct answer in 1-3 sentences, confirmed facts, hypotheses only if needed, targeted fix/check plan, blocking questions only.
- For diagnostics, end each iteration with `Что делать владельцу сейчас`: 1-3 actions, expected output/artifact, and why it removes uncertainty.
- After each new log, dump, or server output, update `Журнал расследования`: `done`, `pending`, `next`. Do not repeat closed points without new facts.
- Discuss behavior in UX terms first: what the user does, sees, and expects. If UI does not change, write `UI не меняется` and state the user-visible or operational effect.
- If a solution is disputed or complex, agree on the user-visible result first, then choose the technical implementation.
- For multi-stage work, define gate criteria: current step, verification, and who confirms the transition. Do not start the next stage before the gate passes.
- If an extra action may be needed but was not explicitly requested, ask before doing it.
- After the owner clarifies a task, restate the understanding, list exact planned edits, and wait for `подтверждаю`.
- During work, give short progress updates. After work, report what changed, where, why, and how it was checked.

### Plan Continuity Before Gates

- Never present a gate, decision, or "need to choose" note as the whole next step.
- Before naming a gate, reread the active roadmap/DoD source: `docs/PROJECT.md` and current numbered working docs in `docs/`, when they exist.
- First state the next planned work from the active docs: stage, exact tasks, affected files/modules when known, and expected user-visible or operational result.
- A gate is valid only when it blocks those named next tasks.
- If active docs do not define concrete next tasks, say that explicitly and propose 2-3 concrete next actions before asking for a decision.
- If owner input is needed, provide: `Next by plan`, `Blocked by`, `Recommendation`, and `Owner action`.
- If the choice is internal and does not affect UX, cost, deployment, security, data durability, or owner workflow, choose conservatively, document the assumption, and continue after required confirmation.
- Do not imply development is stopped unless the stopping condition and unblock action are explicit.

## 6) Dependency Audit And Goal Check

Before edits, do a concise `Dependency Audit`:
- affected UX scenarios and modules;
- data, limits, status, and navigation dependencies;
- possible side effects and regressions;
- UX or logical contradictions by priority.

Do not edit until critical contradictions for the current task are closed. Each change must directly serve the current goal and have a measurable result.

## 7) Server Diagnostics And `comm`

- The working system may run on a server. Server data and processes are the valid diagnostic source for production issues.
- Before writing diagnostic `comm`, read the server/runtime facts in `docs/PROJECT.md` first. Check at least `## Stack`, `## Deployment`, `## Checks`, `## Open Questions`, and any project-specific sections named `Server`, `Runtime`, `Production`, `Staging`, `Diagnostics`, or `Tools`.
- When relying on `docs/PROJECT.md`, cite only the exact relevant line references in owner-facing text, e.g. `docs/PROJECT.md:42`, instead of re-summarizing the whole profile.
- If `docs/PROJECT.md` already confirms available server tools or runtime shape, such as Docker Compose, logs command, SQL access, SQLite/PostgreSQL, `curl`, `make`, `systemctl`, or service names, use those facts directly. Do not ask the owner to run discovery for already documented facts.
- If server facts are missing or too thin, the first diagnostic `comm` should be a one-time read-only runtime inventory for the project, not a narrow discovery for only the next request. Inventory should cover common operational tools and access paths likely to be needed: shell basics, `docker`/`docker compose`, logs access, compose project/services, database client/access method, `curl`/HTTP tools, `make`, `systemctl`/service manager if relevant, repo path, and environment file presence without printing secrets.
- After the owner returns inventory output, record stable server facts in `docs/PROJECT.md` before relying on them in later diagnostics. Do not repeat the same discovery unless a command is missing, the environment changed, or the current task needs a new class of tools not previously checked.
- For server diagnostics, overwrite `comm` in repo root. This does not require separate confirmation.
- `comm` is not a script by default; it is a copy/paste field for manual SSH commands.
- The owner copies the whole `comm` content into SSH and returns output in chat.
- Conclusions about production diagnostics must be based on returned server output.

Server command pacing:
- Before each server-command step, briefly show the exact commands and why they are needed.
- Give error-prone or branching commands one at a time, then wait for owner output before the next command.
- Routine agreed commands with no expected branch may be grouped in a small batch.
- Do not send long command walls where an early failure makes later commands misleading or useless.

Rules for `comm`:
- diagnostic commands only: compact logs, read-only SQL, service/container/image status;
- one command per line, expected run order;
- do not add `echo STEP_*` labels by default;
- keep the first diagnostic pass short, usually 3-6 commands;
- explain in chat what each command group checks, instead of putting explanatory noise into `comm`;
- copy/paste-safe: no placeholders, no broken quotes, no heredoc, no functions, no multi-line script logic;
- each SQL check is one `psql -c "..."` command;
- first request signature/count/top-N summaries, not raw logs;
- normalize volatile fields before dedupe: timestamps, durations, request/correlation IDs, message IDs, account/user/container IDs;
- raw lines are allowed only after summaries, only for targeted drill-down, and only around 50-100 lines per step;
- drill-down must narrow by entity ID, module, event, error signature, and time window when those facts exist.

Never put these in `comm` by default:
- deploy/release/infra changes;
- container restarts or rebuilds;
- git operations;
- migrations, generation, formatting, mass data edits.

If deploy, restart, or environment change is needed, ask the owner explicitly in chat first.

## 8) Observability

- If the first diagnostic pass does not localize the cause, treat it as an observability gap, not a reason to guess.
- Default next step: propose targeted logging or tracing for the problematic path, then implement only after `подтверждаю`.
- Target visibility: one scenario should show `ingress -> handler/service -> internal calls -> external calls -> result`.
- Log both success and failure paths with fields, not separate logic.
- Minimum fields per observed step: start, finish, `elapsed_ms`, result/error, correlation/request ID, and entity IDs needed to link the chain.
- Never log secrets, tokens, passwords, private keys, or raw sensitive data when surrogate fields are enough.

## 9) Large Changes

A change is large if any is true:
- more than 5 files touched, especially 10+;
- more than 3 UX scenarios change;
- API, DB, or critical business logic contracts change;
- 2+ modules get new behavior.

For large changes, create a decision doc first, then implement, then verify documented behavior against actual behavior.

Large-change docs must include: problem/symptoms, as-is, target behavior, boundaries, implementation plan, chosen option rationale, risks/rollback, DoD/checklist, and decision context.

## 10) Documentation Rules

- Every working doc must have `## Checklist`.
- Do not create or fill `README.md` by default. Update it only after the owner explicitly asks for README content.
- Env/config examples must use fenced `dotenv` or shell-style code blocks, one `KEY=value` per line, so variable names are visually distinct from values. Do not write plain unhighlighted env dumps in docs, chat, or `comm`.
- Completion is recorded by checking items to `[x]` after the fact.
- Checklist completion is the primary readiness signal.
- Working docs must not contain vague rules without criteria. If the criterion is unknown, mark it as an open question.
- Safely deferred issues go into a deferred list.
- One active incident has one primary incident doc.
- If root-cause investigation and target redesign run in parallel, keep separate incident and change docs.
- Incident docs track symptoms, confirmed facts, hypotheses, gates, and investigation log.
- Change docs track target behavior, invariants, boundaries, migration plan, risks, and goal metrics.
- If a new fact invalidates an old conclusion, update the primary doc in the same iteration.

## 11) Code Map Defaults

- `cmd/`: app entry points.
- `internal/`: main application logic.
- `api/`: external contracts, when needed.
- `migrations/`: SQL migrations.
- `configs/`: project configs.
- `docs/`: project profile and local working docs.
- `pkg/`: reusable utilities only when reuse is real.

## 12) Do Not Edit Manually

- Generated files are changed through generation commands, not manual edits.
- Root binaries and build artifacts are not source files.
- Vendored or third-party artifacts are not edited unless the owner explicitly chooses that path.
- Canonical generation commands must live in `docs/PROJECT.md`.

## 13) Checks

- General Go tests: `go test ./...` or `make test`.
- If protobuf or generated contracts change: run the project generation command.
- Local service smoke only when relevant: `make up`, `make logs`, `make down`.
