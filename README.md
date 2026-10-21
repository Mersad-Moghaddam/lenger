# Lenger

**Anchor your containers.** An image-first, keyboard-driven Docker terminal app
for Linux, with optional read-only Omarchy theme integration.

> **Coding-stage skeleton, not a verified release.** Builds, tests, lint,
> recordings and gallery generation were intentionally not run. Dependency
> installation encountered a broken Go checksum-cache path. `go.sum`, transitive
> dependency resolution and reviewed golden baselines remain outstanding.

Public repository: **https://github.com/Mersad-Moghaddam/lenger**.
Contributions are welcome; start with [CONTRIBUTING.md](CONTRIBUTING.md).
For security concerns read [SECURITY.md](SECURITY.md). Release and package
distribution gates are documented in [docs/DISTRIBUTION.md](docs/DISTRIBUTION.md).

## Run

Requires Go 1.27+, a terminal of at least 80×24 cells, and optionally a reachable
Docker daemon. Resolve dependencies in a working Go environment first:

```sh
go mod tidy

# Explicit offline demo; never contacts Docker.
LENGER_ENGINE=fake go run ./cmd/lenger --theme midnight

# Real daemon; errors are shown rather than falling back to fake data.
go run ./cmd/lenger --engine docker

# Offline screen matrix, or one theme and viewport.
go run ./cmd/lenger --gallery
NO_COLOR=1 go run ./cmd/lenger --gallery --theme midnight --size 120x40
```

Flags: `--engine docker|fake`, `--theme`, `--size WIDTHxHEIGHT`, `--debug`,
`--gallery`. `LENGER_ENGINE=fake`, `NO_COLOR`, `NO_MOTION=1` and XDG directories
are respected. Debug startup metadata goes to `$XDG_STATE_HOME/lenger/debug.log`
(default `~/.local/state/lenger/debug.log`), not the terminal.

## Keys

| Key | Action |
| --- | --- |
| `1`–`7` | Dashboard, images, and clearly labeled future workspaces |
| `j/k` or arrows | Move through images |
| `/`, `o` | Filter, cycle sort |
| Space, shift-arrows | Select, extend selection |
| `i` / Enter | Inspect image with sensitive values masked |
| `d` | Removal preview; `y` confirms; never forced |
| `p` | Pull an image |
| Esc | Hide overlay without cancelling a pull |
| `J`, Enter | Job history, open latest job |
| `x`, `R` in pull overlay | Cancel, retry |
| `:` / Ctrl+P | Fuzzy action and image palette |
| `?`, `q` | Help, quit |

## Configuration

Read from `~/.config/lenger/config.toml` or `$XDG_CONFIG_HOME/lenger/config.toml`.
The configuration schema is in [PRD Appendix A](docs/PRD.md#appendix-a-example-configuration).
Only a subset is active in this skeleton; see [decisions](docs/DECISIONS.md).
Docker credential helpers are used through Docker's standard config. Lenger
does not persist registry credentials. Proxy configuration for actual image
downloads belongs to the Docker daemon, not just the UI process.

## Checks to run later

```sh
go mod tidy
make build
make test
make lint
make check
make vhs
```

`make check` intentionally fails until missing golden baselines are explicitly
generated and reviewed. Follow [testdata/README.md](testdata/README.md); missing
baselines are not silently accepted. VHS is optional and skipped if absent.

## Scope and documentation

Image listing, safe inspection/removal policy, pull streaming, a fixture daemon,
dashboard counts, command palette, themes and offline gallery code are present.
The remaining MVP and several M0 polish items are not complete. See the precise
status and limitations in [DECISIONS.md](docs/DECISIONS.md), the
[architecture](docs/ARCHITECTURE.md), [UI guide](docs/UX.md) and
[source-of-truth PRD](docs/PRD.md).

MIT licensed. Linux amd64/arm64 release configuration is provided, but publishing
is disabled and no artifacts have been built.
