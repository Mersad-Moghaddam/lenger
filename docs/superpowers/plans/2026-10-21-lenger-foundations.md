# Lenger Foundations Implementation Plan

> **For agentic workers:** Use executing-plans to implement this plan inline. Verification execution is explicitly disabled by the user for this session.

**Goal:** Implement M0 and an images/pull vertical slice without claiming verification.

**Architecture:** Engine adapters expose context-aware domain types to services. Bubble Tea owns UI state; asynchronous commands deliver snapshots, events and job progress. Semantic tokens drive shared rendering and an offline gallery.

**Tech Stack:** Go, Charm v2, Moby client, TOML, Harmonica, fuzzy matching.

**Spec:** `docs/PRD.md`, especially sections 7, 9 and 11.

## Global Constraints
- No builds, tests, lint runs, recordings or other heavy execution in this session.
- UI never imports the Docker adapter; all I/O runs outside Update.
- No credentials or unmasked environment values in logs or default inspection.
- No edits to desktop/system configuration, no publishing.
- Colors and spacing originate in semantic tokens.

## Review Focus
- Cancellation during stream delivery must release producers and HTTP bodies.
- Permission failures must not silently select a fake daemon.
- Terminal resize and long untrusted resource names must stay bounded.
- In-use images must not be force-removed.
- Secret environment values and labels must stay masked.

## Tasks / execution ledger
- [ ] 1. Foundation: module, Makefile, CI, release configuration, package documentation.
- [ ] 2. Engine: domain interface, classified errors, scripted fake, Docker adapter with bounded streams and reconnect.
- [ ] 3. Core: normalized store, services, cancellable pull job aggregation, TOML configuration.
- [ ] 4. Design: tokens, three bundled themes, read-only Omarchy adapter, charts, shared components, animation and hit-testing.
- [ ] 5. App: async bootstrap, dashboard, images list/details, filters/sort/selection, confirmation, pull overlay, help/palette, input and event routing.
- [ ] 6. Delivery: flags, offline gallery, authored tests, VHS tapes and accurate documentation.

### Session ledger

All six tasks have authored source/configuration/documentation, but remain
unchecked because executable acceptance was explicitly prohibited. Outstanding
scope and exact requirement status are listed in `docs/DECISIONS.md`.

- Lightweight preflight: Go version reported 1.27.3; Docker socket permission
  denied; default Omarchy theme directory absent.
- Dependency setup: both attempts failed checksum-cache lookup; no checksums
  were bypassed and no successful installation is claimed.
- Formatter: absent from PATH and reported GOROOT, so formatting remains pending.
- No build, test, vet, lint, gallery, recording or performance command was run.
- No per-task verified commits exist; source is committed as an explicitly
  unverified coding-stage batch rather than manufacturing green milestones.

## Interfaces
- Engine methods take `context.Context`; pull calls a context-aware progress callback and returns an error.
- `services.Service` is the sole engine facade consumed by the app; engine construction lives in the entrypoint.
- `jobs.Manager` owns pull contexts, session history and immutable snapshots; app waits on a notification channel.
- `tokens.Palette` contains all role colors; `style.Sheet` and components consume it.
- `app.New(ctx, service, config, palette)` creates an idle model; `gallery` renders models without starting I/O.

## Verification (deferred by user)
Author unit/model/layout/contrast/token tests and a full-program smoke test, but do not execute them. Do not generate or accept golden frames without a reviewed render. Future verification: `go mod tidy`, `make check`, `make gallery`, real-daemon smoke test, VHS recordings.

## Rulings
- User's coding-only instruction overrides PRD self-verification loop and RED/GREEN execution; tests may be authored but remain unverified.
- No existing Git repository or origin: use `github.com/OWNER/lenger`. Initialize a repository only after source work; commits must explicitly state verification was skipped.
- No graph index exists for this empty project; filesystem inspection is the fallback. No indexing job is started in this coding-only session.
- Use Go proxy metadata for latest versions and dependency source for API inspection rather than compiling tools.
