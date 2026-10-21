# Contributing to Lenger

Lenger is an early, unverified Go TUI skeleton. Do not use it for critical
container workloads or assume a green CI status that has not been demonstrated.

## Start here

Read `docs/ARCHITECTURE.md` for boundaries, `docs/UX.md` for rendering conventions,
and `docs/DECISIONS.md` for known gaps. `docs/PRD.md` is the product source of truth.

Keep pull requests focused on one behavior. Add regression tests for fixes,
avoid new dependencies where the standard library suffices, and explain policy
changes (especially deletion, credentials, or remote-daemon selection).

## Development commands

```sh
go mod tidy
gofmt -w cmd internal
make build
make test
make lint
```

Gallery baselines are not yet committed. See `testdata/README.md` before
establishing them. Never update snapshots simply to hide a rendering regression.
CI is manually dispatched during this coding-only stage; it is not evidence of
release readiness. Real Docker integration needs an explicitly authorized,
disposable daemon and must never target a contributor's production containers.

## Code conventions

- Keep one responsibility per file; prefer readable multi-line control flow.
- Standard-library imports, third-party imports, then project imports.
- Capture command dependencies on the model thread before returning a command.
- Never read changing model fields inside asynchronous commands.
- Honour context cancellation and deadlines at I/O boundaries.
- Keep secrets out of messages, error output, logs and fixtures.
- Do not force deletions or silently change daemon/desktop configuration.
