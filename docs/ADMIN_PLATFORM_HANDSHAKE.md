# Admin Platform Handshake

Status: reusable admin-integration standard for future projects.

Purpose: define how a project should be prepared so it can be managed from the shared owner admin platform without copying full admin panels or coupling the platform to project internals.

## Boundary

Default shape:

```text
Owner browser
  -> shared admin platform
    -> private project admin contract
      -> project business logic
```

Rules:
- the browser talks to the shared admin platform only;
- the shared admin platform owns operator UI, login policy, shared permissions, shared workflows, and admin audit;
- each project owns its own business rules and exposes explicit private admin commands;
- project services validate calls from the shared admin platform before executing admin commands;
- the shared admin platform must not write directly into project databases for business actions;
- hidden URLs are not security.

## Network

First practical deployment:
- all services may run on one server;
- use a private Docker Compose network;
- use internal HTTP/JSON for the first contract;
- do not expose project admin endpoints as public browser APIs.

Future deployment:
- keep the contract stable while changing only connector config and transport;
- acceptable private transports include private HTTPS, VPN/WireGuard, Tailscale, mTLS, and later gRPC/protobuf;
- remote projects still validate admin-platform calls and enforce their own business rules.

## Minimum Contract

Recommended first version:
- `GET /admin/v1/health`: service availability;
- `GET /admin/v1/capabilities`: supported resources, reads, commands, events, jobs, and optional UI hints;
- MVP resource reads, such as users, orders, campaigns, settings, or project-specific entities;
- explicit command endpoints for business actions, such as grant/revoke flags, block/unblock, resend, recalculate, or enqueue delivery;
- event/timeline reads for project events and user events;
- job create/status endpoints for long-running operations.

Example capability response shape:

```json
{
  "project": "example",
  "version": "admin.v1",
  "resources": ["users", "events"],
  "actions": ["users.grant_flag", "users.revoke_flag"],
  "jobs": ["broadcast.enqueue"],
  "events": ["user.timeline", "project.errors"]
}
```

## Command Rules

Every mutating admin command should have:
- server-side authorization;
- project-side validation of the admin-platform caller;
- input validation;
- timeout and cancellation;
- audit intent fields;
- observable success and failure result;
- no secret logging.

Long operations should return a job ID and expose status instead of holding the request open.

## Project Intake

At project start, record:
- whether the project will be managed from the shared owner admin platform;
- MVP admin-visible entities and workflows;
- deferred admin workflows, separately from MVP;
- private admin contract version, transport, and network scope;
- first `health` and `capabilities` response;
- first admin reads, commands, events, and long-running jobs;
- how the project validates admin-platform calls, without storing secrets in docs.

## Checklist

- [x] Shared admin boundary recorded.
- [x] Same-server first transport recorded.
- [x] Future remote transport direction recorded.
- [x] Minimum private contract recorded.
- [x] Capability-first approach recorded.
- [x] Direct browser/project admin calls forbidden.
- [x] Direct shared-admin database writes forbidden for business actions.
