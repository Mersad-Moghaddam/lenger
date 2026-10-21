# Architecture

The product architecture and rationale live in [PRD section 9](PRD.md#9-system-design).
This document describes the implemented package boundaries, not a second PRD.

## Package map

| Package | Responsibility |
| --- | --- |
| `cmd/lenger` | Flags, engine construction, logging, terminal program lifecycle |
| `internal/engine` | SDK-independent resources, streaming contracts, error classification |
| `internal/engine/docker` | Moby client, credential-store lookup, progress decoding, event reconnect |
| `internal/engine/fake` | Isolated, cancellable in-memory fixture daemon |
| `internal/core/services` | Concurrent snapshots, safe inspection, reference validation, removal policy |
| `internal/core/store` | Normalized resource batch and image/container usage index |
| `internal/core/jobs` | Cancellable pulls, progress aggregation and session history |
| `internal/config` | XDG TOML configuration, defaults and validation |
| `internal/ui/tokens` | Semantic colors, shared cell geometry, icon vocabulary |
| `internal/ui/theme` | Bundled palettes, contrast utilities and perceptual gradient blending |
| `internal/ui/style` | Token-derived styles, sanitization and bounded rendering |
| `internal/ui/chart` | Braille area charts, sparklines, eighth-cell bars and proportional segments |
| `internal/ui/components` | Tabs, panels, pills, overlays, pull progress, help and formatting |
| `internal/ui/views` | Dashboard, responsive images master/detail and labeled deferred screens |
| `internal/ui/anim` | Finite Harmonica spring primitive; not yet wired to visual transitions |
| `internal/ui/zones` | Frame-local rectangle hit-testing primitive |
| `internal/app` | Router, keyboard/mouse input, messages, overlays and one composed frame |
| `internal/platform/omarchy` | Read-only optional palette loading/watching and setup guidance |
| `internal/gallery` | Offline fixture matrix, shared production rendering, layout/golden tests |

## Conventions

- Only the entrypoint constructs the Docker adapter. App code uses services.
- `Update` handles messages only. Engine and filesystem operations run in commands.
- Model state is owned by the Bubble Tea loop. Job state has its own mutex;
  snapshots detach layer slices before rendering.
- A session context cancels event watchers, polling and workers. Pull response
  bodies close on every return path. Stream delivery selects on cancellation.
- Destructive operations recheck container references and never use force.
- Inspection conservatively masks every environment and label value, command
  argument and image-history command. No secret-reveal action exists yet.
- Unknown/deferred capabilities are explicit, never simulated against real Docker.
- Real errors are presented as safe classifications; raw registry failures are
  neither shown nor logged because URLs and daemon messages can contain secrets.

## Unverified integration boundaries

See [DECISIONS.md](DECISIONS.md) for the coding-only execution constraint. In
particular, dependency resolution, compilation, Moby event reconnect behavior,
terminal interaction and visual baselines still need executable verification.
