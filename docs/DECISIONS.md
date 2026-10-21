# Decisions and implementation ledger

## 2026-10-21

- Source of truth: `PRD.md`. Scope is M0 plus images/pull, not the full MVP.
- User explicitly requested coding only: no builds, test runs, lint runs,
  generated snapshots, recordings or performance runs. Acceptance criteria
  remain pending, not waived. Authored tests are not evidence of passing tests.
- No origin exists; module path is `github.com/OWNER/lenger` until ownership
  is selected. Name availability has not been verified; publishing is disabled.
- Installed Go is 1.27.3. Docker client 29.9.1 is present, but socket access
  is denied. Default real mode must show that failure; fake mode is explicit.
- `~/.config/omarchy/current/theme` is absent. Adapter supports `colors.toml`,
  Ghostty and Alacritty theme palettes read-only, falling back to Midnight.
- Latest dependency versions are discovered from the Go module proxy. SDK
  source inspection replaces compilation-based API discovery in this session.
- No personal/system configuration is changed. No service or daemon is started.
- Prefer no idle animation to a perpetual heartbeat, resolving PRD 7.7's idle
  scheduling constraint in favor of zero decorative timers.

### Coding-stage status and deviations

- `go get ...@latest` was attempted, then retried with a session-local GOPATH.
  Both failed on `/go/pkg/sumdb/sum.golang.org/latest`. Checksum verification
  was not disabled. Versions are pinned from proxy metadata, but `go.sum` and
  transitive requirements need `go mod tidy` in a working Go environment.
- New Moby client v0.6.2 uses result wrappers (`Items`) and defaults to API
  negotiation. Adapter code follows inspected upstream source, not old SDK APIs.
- Foundation, engine/core, UI and delivery code are authored, not verified.
  No milestone acceptance criteria are claimed met. No golden text is fabricated.
- Conservative secret policy masks all environment/label values and history
  commands, not just names resembling secrets. Entrypoint and command arguments
  are hidden too. Reveal and collapsible inspect trees are deferred.
- Local Docker contexts and SSH transport (PLT-04) are explicitly rejected rather
  than accidentally targeting another daemon. DOCKER_HOST unix/tcp and Docker
  TLS environment are delegated to Moby. Default selected CLI context is not
  loaded yet. Context-aware helper subprocess execution is still required:
  Docker CLI's credential-store API does not accept context cancellation.
- SDK uses standard proxy transport behavior; actual image-download proxies
  must be configured on the daemon. No daemon config is edited.
- Initial service facade implements image operations and snapshots only; future
  container/project/registry/cleanup services return explicit deferred status via
  the common typed error helper rather than exposing misleading operations.
- Event reconnect uses a timestamp replay cursor; exact duplicate-event
  suppression and a reconnect-without-events full refresh need hardening.
- Pull job completion aggregates known bytes, not extraction completion. Speed
  is a session average; download/extraction phase accounting needs refinement.
- Job panic recovery marks failure without logging panic payloads that might
  contain secrets. Dedicated safe crash reports remain pending.
- Full UI motion is deferred; the finite spring is implemented but not wired to
  visual movement. No invisible/decorative motion timers are scheduled.
- Density toggle state is captured but density-specific layout is deferred.
  Hit-test primitive exists; mouse tab/row handling is basic, no dragging yet.
- Gallery covers seven screens, help, palette, pull, removal and jobs; it is not
  yet a complete standalone component catalog or cleanup preview.
- Golden tests fail when baselines are absent. CI is scaffolding, not green.
  Docker-in-CI integration, event-drop chaos tests and full job/store coverage
  remain to be authored. Existing tests have not been executed.
- Formatting was attempted without compiling anything: `gofmt` was absent from
  PATH and `/usr/local/go/bin/gofmt` (the reported GOROOT). Sources therefore
  still need `gofmt -w cmd internal` once the Go installation is repaired.

### Requirement status

| Requirement | Coding status |
| --- | --- |
| IMG-01 | List/filter/name-size-age sort/usage and selection authored; table tuning pending |
| IMG-02 | Pull/layer progress/cancel/retry/hide authored; phase accounting and credentials cancellation pending |
| IMG-03 | Bulk preview and non-forced reference-checked removal authored |
| IMG-04 | Masked structured detail authored; scroll/reveal/tree pending |
| IMG-05 | Deferred to M1 |
| CTR-01–CTR-05 | Counts and usage snapshots only; full container workflows deferred |
| VOL-01, SYS-01, SYS-02 | Deferred; dashboard image-byte summary is not a full disk-usage/cleanup implementation |
| PLT-01 | Fuzzy actions/images with command preview; highlights/recents pending |
| PLT-02 | Image filter only; saved/global filters pending |
| PLT-03 | Space and shift-arrow extension authored; arbitrary range anchors pending |
| PLT-08 | Pull jobs/session history authored; other job kinds pending |
| REG-01 | Error classification authored; unverified against real failures |
| REG-04 | Credential-store/proxy environment path authored; helper cancellation pending |
| PRJ-01–PRJ-05, CTR-06–CTR-07, NET-01, SYS-03–SYS-04 | M2; event feed foundation only |
| IMG-06–IMG-11, REG-02–REG-03, REG-05–REG-06 | M3/out of scope; basic history ribbon only |

### Next hand-off

Implement milestone M1 as defined in `docs/PRD.md` section 11.1, following
`docs/ARCHITECTURE.md` and `docs/UX.md`. First repair dependency setup, resolve
modules without disabling checksums, compile and address integration errors,
review gallery frames and establish baselines. Finish outstanding M0 items
listed above, then deliver IMG-05, CTR-01–CTR-05, VOL-01, SYS-01/SYS-02, and
complete PLT-01–PLT-03/PLT-08 and REG-01/REG-04. Once execution is authorized,
run `make check` and real-Docker lifecycle integration tests, recording actual
results rather than inferring success from authored code.
