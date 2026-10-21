# Working on Lenger

Read `docs/PRD.md` completely, then `docs/ARCHITECTURE.md`, `docs/UX.md` and
`docs/DECISIONS.md` before changing code.

- Build: `make build`
- Run offline: `LENGER_ENGINE=fake make run`
- Tests: `make test`
- Lint: `make lint`
- Full checks: `make check`
- Gallery: `make gallery`

Keep adapters behind services, I/O out of Update, colors/spacing in tokens,
and exported identifiers documented. TODOs must name a milestone.

The initial coding session explicitly did not execute builds or tests.
