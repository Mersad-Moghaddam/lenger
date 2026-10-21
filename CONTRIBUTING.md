# Contributing to Lenger

Lenger is a Docker TUI with offline fixtures and automated checks. Test changes
against disposable workloads; do not assume that fixtures or a prior green CI
run verify a different commit or production Docker integration.

## Start here

Read `docs/ARCHITECTURE.md` for boundaries, `docs/UX.md` for rendering conventions,
and `docs/DECISIONS.md` for known gaps. The approved redesign in
`docs/superpowers/specs/2026-10-21-lenger-redesign.md` defines the container-first
four-resource / five-inspector target; older product docs may describe prior UI.

Keep pull requests focused on one behavior. Add regression tests for fixes,
avoid new dependencies where the standard library suffices, and explain policy
changes (especially deletion, credentials, or remote-daemon selection).

## Development commands

```sh
go mod download
go fmt ./cmd/... ./internal/...
make build
make test
make lint
```

Gallery baselines are committed under `testdata/golden`. Render and review the
gallery before intentionally updating them with
`go test ./internal/gallery -run TestGolden -update`; never update snapshots
simply to hide a rendering regression. Run `make check` without update flags.
CI runs automatically for main pushes and pull requests, supports manual runs,
and is reused by the tag-release verification gate. A configured workflow is not
evidence of a successful run; inspect results for the exact commit. Real Docker
integration needs an explicitly authorized,
disposable daemon and must never target a contributor's production containers.

## Code conventions

- Keep one responsibility per file; prefer readable multi-line control flow.
- Standard-library imports, third-party imports, then project imports.
- Capture command dependencies on the model thread before returning a command.
- Never read changing model fields inside asynchronous commands.
- Honour context cancellation and deadlines at I/O boundaries.
- Keep secrets out of messages, error output, logs and fixtures.
- Do not force deletions or silently change daemon/desktop configuration.
