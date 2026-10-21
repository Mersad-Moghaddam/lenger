# Lenger Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign Lenger from scratch into a high-performance, calm Docker TUI featuring a 35/65 Master-Detail container cockpit, evidence-grounded diagnostics, bounded virtualized logs, responsive layouts, and lean multi-resource navigation.

**Architecture:**
A Bubble Tea v2 terminal architecture where `1: Containers` acts as the primary Master-Detail view (expanding on demand from a 100% list into a 35% list / 65% 5-tab inspector drawer with resizable divider), while `2: Images`, `3: Volumes`, and `4: Networks` provide clean, single-table resource management. Container list context is strictly preserved across view switches while background log/stats goroutines are suspended when closed. Diagnostics are segregated into three traceable evidence tiers (Verified Facts, Observed Telemetry, Hypotheses).

**Tech Stack:**
Go 1.24+, Charm Bubble Tea v2, Lipgloss v2, Docker Engine Go SDK (`github.com/moby/moby/client`), `container/ring`, `golang.org/x/term`.

**Spec:** `docs/superpowers/specs/2026-10-21-lenger-redesign.md`

## Global Constraints
- Target release: `v0.2.0`.
- Minimum supported terminal size: 80×24 cells.
- Responsive breakpoint: $\ge 87$ columns renders Side-by-Side split; $< 87$ columns renders Slide-Over Overlay.
- Minimum list width: 32 columns; minimum inspector width: 54 columns.
- Default split ratio: 35% list / 65% inspector (divider adjustable via `[` and `]`).
- Bounded log capacity: 10,000 lines maximum per active container buffer.
- Sensitive environment variable auto-remask timeout: 60 seconds of inactivity, container switch, or drawer closure.
- Telemetry gathering must be performed exclusively via Docker Engine API (`ContainerStats`) to guarantee 100% compatibility with remote `DOCKER_HOST`, macOS, WSL2, and rootless environments.
- Zero goroutine leaks: closing the drawer or switching away from Containers must immediately cancel active log streaming and stats polling contexts.

## Review Focus
1. **Container has no health check configured:** `ContainerInspection.HealthCheck` is nil; Health tab and tab latching must treat it as standard running without health check, never panic or display false failures. (*Tested in Task 5*)
2. **Container uses unsupported or non-reader logging driver (`none`, `syslog`, `journald`):** `ContainerLogs` returns error; Live Logs tab must display a clear, informative notice instead of crashing or showing an empty black screen. (*Tested in Task 4 & Task 6*)
3. **Container exited before Lenger started:** Docker `ContainerStats` is unavailable for dead containers; Metrics and Health tabs must explicitly report telemetry as `Unavailable (container exited prior to session)` rather than displaying false zero values. (*Tested in Task 4 & Task 7*)
4. **Terminal window shrinks below 87 columns while drawer is open:** Layout must seamlessly transition to slide-over overlay mode without losing active tab, cursor, or scroll position. (*Tested in Task 2 & Task 11*)
5. **Sensitive environment variable contains mixed casing or common token suffixes (`API_Secret`, `DB_PASSWD`, `Auth_Token`):** Case-insensitive regex matching must catch and mask all secret variants by default. (*Tested in Task 8*)

---

## File Structure & Module Map

```
internal/
├── ui/
│   ├── masterdetail/             # Master-detail layout engine
│   │   ├── layout.go             # Split ratio, breakpoint, min-width clamp, divider
│   │   └── layout_test.go
│   ├── views/
│   │   ├── containers/           # Containers list with compose stack grouping
│   │   │   ├── list.go
│   │   │   └── list_test.go
│   │   └── inspector/            # 5-Tab Diagnostic Cockpit
│   │       ├── inspector.go      # Tab bar, latching logic, zoom manager
│   │       ├── tab_health.go     # 3-tier evidence provenance, previous logs trigger
│   │       ├── tab_logs.go       # 10k bounded ring buffer, follow toggle, search
│   │       ├── ringbuffer.go     # Thread-safe ring buffer implementation
│   │       ├── tab_metrics.go    # Gauges, CPU delta calculator, session sparklines
│   │       ├── tab_env.go        # Sensitive key masking, 60s timeout, port modal
│   │       ├── tab_timeline.go   # Chronological event aggregator, integrity notice
│   │       └── inspector_test.go
│   └── components/
│       ├── confirm.go            # Destructive action confirmation modals
│       └── tabs.go               # Top navigation 1-4
├── engine/
│   ├── engine.go                 # Core domain models (extended LogOptions with Until)
│   └── docker/
│       ├── logs.go               # ContainerLogs with Until & driver error handling
│       └── stats.go              # Stats polling with CPU delta calculation
└── app/
    ├── model.go                  # Bubble Tea state, context preservation, lifecycle
    ├── focus.go                  # Keyboard focus manager (List <-> Drawer <-> Controls)
    ├── view.go                   # Master render pipeline
    ├── actions.go                # Confirmed destructive actions
    └── integration_test.go       # End-to-end acceptance tests
```

---

### Task 1: Navigation Re-indexing & Context Preservation Engine

**Files:**
- Modify: `internal/ui/components/components.go:15-30`
- Modify: `internal/app/model.go:190-230`
- Modify: `internal/app/input.go:40-90`
- Modify: `internal/app/view.go:50-86`
- Test: `internal/app/navigation_test.go`

**Interfaces:**
- Consumes: `components.TabNames() []string`, `Model.tab int`
- Produces: `TabNames()` returning `[]string{"Containers", "Images", "Volumes", "Networks"}`
- Produces: `Model.switchTab(target int)` preserving Container filter, cursor, and scroll offset while canceling active drawer streams.

- [ ] **Step 1: Write the failing navigation test**
Write `internal/app/navigation_test.go` asserting that `TabNames()` equals `[]string{"Containers", "Images", "Volumes", "Networks"}`, that default `tab == 0` is Containers, that switching from tab 0 to tab 1 suspends drawer streaming contexts, and that switching back to tab 0 restores container cursor and filter.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/app -run TestNavigation`
Expected: FAIL due to mismatched tab names and missing context preservation.

- [ ] **Step 3: Update `TabNames()` and navigation routing**
In `internal/ui/components/components.go`, update `TabNames()` to return `[]string{"Containers", "Images", "Volumes", "Networks"}`. In `internal/app/model.go`, re-map tab numbers (`0: Containers`, `1: Images`, `2: Volumes`, `3: Networks`). Implement `switchTab(target int)` in `internal/app/input.go` that stores container cursor/filter, cancels drawer log/stats contexts when leaving tab 0, and restores them when returning to tab 0.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/app -run TestNavigation`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/components/components.go internal/app/model.go internal/app/input.go internal/app/view.go internal/app/navigation_test.go
git commit -m "feat(app): re-index navigation to 1-4 and preserve container context across tabs"
```

---

### Task 2: Master-Detail Layout Engine & Responsive Split

**Files:**
- Create: `internal/ui/masterdetail/layout.go`
- Create: `internal/ui/masterdetail/layout_test.go`

**Interfaces:**
- Consumes: Terminal width, height, drawer open state, divider ratio.
- Produces:
  ```go
  type LayoutMode int
  const (
      ModeListOnly LayoutMode = iota
      ModeSplit
      ModeOverlay
  )

  type Layout struct {
      Mode           LayoutMode
      ListWidth      int
      InspectorWidth int
      Height         int
      Ratio          float64 // default 0.35
  }

  func ComputeLayout(totalWidth, totalHeight int, drawerOpen bool, ratio float64) Layout
  func (l *Layout) AdjustRatio(delta float64)
  ```

- [ ] **Step 1: Write the failing layout tests**
Write `internal/ui/masterdetail/layout_test.go` testing:
- When `drawerOpen == false`, `Mode == ModeListOnly`, `ListWidth == totalWidth`.
- When `drawerOpen == true` and `totalWidth >= 87`, `Mode == ModeSplit`, clamping `ListWidth >= 32` and `InspectorWidth >= 54`.
- When `drawerOpen == true` and `totalWidth < 87` (e.g. 80 cols), `Mode == ModeOverlay`, `InspectorWidth == totalWidth`.
- Review Focus 4: Resizing width across the 87 boundary dynamically switches between `ModeSplit` and `ModeOverlay`.
- `AdjustRatio` clamped between 0.20 and 0.50.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ui/masterdetail`
Expected: FAIL (package does not exist).

- [ ] **Step 3: Implement `internal/ui/masterdetail/layout.go`**
Implement `ComputeLayout` enforcing the 87-column responsive threshold, min list width 32, min inspector width 54, default ratio 0.35, and ratio adjustments.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ui/masterdetail`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/masterdetail/layout.go internal/ui/masterdetail/layout_test.go
git commit -m "feat(ui): add masterdetail responsive layout engine with 87-col breakpoint"
```

---

### Task 3: Keyboard Focus Hierarchy & Drawer Lifecycle Manager

**Files:**
- Create: `internal/app/focus.go`
- Create: `internal/app/focus_test.go`
- Modify: `internal/app/input.go`

**Interfaces:**
- Consumes: `tea.KeyMsg`, current focus state, drawer open state.
- Produces:
  ```go
  type FocusArea int
  const (
      FocusList FocusArea = iota
      FocusDrawerTabs
      FocusDrawerControls
  )

  type FocusManager struct {
      Area        FocusArea
      ActiveTab   int // 0..4
      ControlIdx  int
      ControlMax  int
      DrawerOpen  bool
      Zoomed      bool
  }

  func (f *FocusManager) HandleKey(key string) (handled bool)
  func (f *FocusManager) OpenDrawer(initialTab int)
  func (f *FocusManager) CloseDrawer()
  func (f *FocusManager) ToggleZoom()
  ```

- [ ] **Step 1: Write the failing focus tests**
Write `internal/app/focus_test.go` asserting:
- Initial state: `Area == FocusList`, `DrawerOpen == false`.
- Pressing `Enter` in list calls `OpenDrawer(tab)`, sets `DrawerOpen = true`, focus remains on `FocusList` or moves cleanly.
- Pressing `Tab` cycles: `FocusList -> FocusDrawerTabs -> FocusDrawerControls -> FocusList`.
- In `FocusDrawerTabs`: `left`/`right` changes `ActiveTab` in range 0..4.
- In `FocusDrawerControls`: `Tab`/`Shift+Tab` increments/decrements `ControlIdx`.
- Pressing `Esc` when `Zoomed == true` sets `Zoomed = false` (leaves drawer open).
- Pressing `Esc` when `Zoomed == false` sets `DrawerOpen = false` and returns focus to `FocusList`.
- Pressing `z` toggles `Zoomed`.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/app -run TestFocus`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/app/focus.go`**
Implement the two-pane hierarchical focus manager and integrate with `internal/app/input.go`.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/app -run TestFocus`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/app/focus.go internal/app/focus_test.go internal/app/input.go
git commit -m "feat(app): implement accessible keyboard focus hierarchy and drawer lifecycle"
```

---

### Task 4: Docker Engine Enhancements: Graceful Telemetry & Previous Logs

**Files:**
- Modify: `internal/engine/engine.go:100-115`
- Modify: `internal/engine/docker/logs.go:18-50`
- Modify: `internal/engine/docker/container.go`
- Modify: `internal/engine/fake/logs.go`
- Modify: `internal/engine/fake/container.go`
- Create: `internal/engine/docker/logs_test.go`
- Create: `internal/engine/docker/stats_test.go`

**Interfaces:**
- Consumes: Docker client `ContainerLogs`, `ContainerStats`.
- Extends:
  ```go
  type LogOptions struct {
      Follow   bool
      Tail     int
      Until    time.Time // Added for previous-instance logs
      Previous bool
  }
  ```
- Produces:
  ```go
  var ErrLogsNotSupported = errors.New("logging driver does not support reading logs")

  type TelemetryState int
  const (
      TelemetryActive TelemetryState = iota
      TelemetrySampled
      TelemetryUnavailable
  )
  ```

- [ ] **Step 1: Write failing tests for Until logs and stats telemetry**
Write `internal/engine/docker/logs_test.go` testing that `LogOptions.Until` formats RFC3339 correctly, and that an unsupported driver error returns `ErrLogsNotSupported`.
Write `internal/engine/docker/stats_test.go` testing that CPU percentage calculation handles zero interval without divide-by-zero, and that dead containers return `TelemetryUnavailable` with no error crash.

- [ ] **Step 2: Run tests to verify they fail**
Run: `go test -v ./internal/engine/docker`
Expected: FAIL.

- [ ] **Step 3: Implement Until logs and graceful telemetry handling**
In `internal/engine/engine.go`, add `Until time.Time` and `Previous bool` to `LogOptions`. In `internal/engine/docker/logs.go`, pass `Until` to `client.ContainerLogsOptions` and wrap driver reading errors into `ErrLogsNotSupported`. In `internal/engine/docker/stats.go` (and fake engine), calculate CPU delta safely and mark dead container stats as unavailable.

- [ ] **Step 4: Run tests to verify they pass**
Run: `go test -v ./internal/engine/docker`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/engine/engine.go internal/engine/docker/logs.go internal/engine/docker/container.go internal/engine/fake/logs.go internal/engine/fake/container.go internal/engine/docker/logs_test.go internal/engine/docker/stats_test.go
git commit -m "feat(engine): add log Until time filter, driver error wrapping, and graceful stats telemetry"
```

---

### Task 5: Health & Diagnostics Drawer Tab (3-Tier Provenance)

**Files:**
- Create: `internal/ui/views/inspector/tab_health.go`
- Create: `internal/ui/views/inspector/tab_health_test.go`

**Interfaces:**
- Consumes: `engine.ContainerInspection`, `sheet style.Sheet`, session stats peak memory.
- Produces:
  ```go
  type InitialTab int
  const (
      TabHealth InitialTab = 0
      TabLogs   InitialTab = 1
      TabMetrics InitialTab = 2
      TabEnv    InitialTab = 3
      TabTimeline InitialTab = 4
  )

  func DetermineInitialTab(c engine.ContainerInspection) InitialTab
  func RenderHealthTab(s style.Sheet, c engine.ContainerInspection, peakMemory uint64, peakObserved bool, w, h int) string
  ```

- [ ] **Step 1: Write failing tests for latching logic and 3-tier provenance**
Write `internal/ui/views/inspector/tab_health_test.go` testing:
- Tab latching table: `Exited (137)` $\rightarrow$ `TabHealth`; `Running + Unhealthy` $\rightarrow$ `TabHealth`; `Running + Healthy` $\rightarrow$ `TabLogs`; `Running + Nil HealthCheck` $\rightarrow$ `TabLogs`; `Paused` $\rightarrow$ `TabHealth`.
- Review Focus 1: `HealthCheck == nil` renders cleanly without panicking and reports standard container state.
- Tier 1: Verified facts (`OOMKilled: true`, `ExitCode: 137`) render under `[Docker Engine Inspect]` provenance header.
- Tier 2: Peak memory renders if `peakObserved == true`; renders `Unavailable (container exited prior to telemetry session)` if `peakObserved == false`.
- Tier 3: Hypotheses are explicitly prefixed with `Hypothesis:` and action buttons `[p] View Previous Instance Logs`, `[r] Restart`.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ui/views/inspector -run TestHealth`
Expected: FAIL.

- [ ] **Step 3: Implement `DetermineInitialTab` and `RenderHealthTab`**
Implement `internal/ui/views/inspector/tab_health.go` strictly segregating Tier 1, Tier 2, and Tier 3 diagnostics with Lipgloss styling and action button anchors.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ui/views/inspector -run TestHealth`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/views/inspector/tab_health.go internal/ui/views/inspector/tab_health_test.go
git commit -m "feat(inspector): add health and diagnostics tab with 3-tier evidence provenance"
```

---

### Task 6: Bounded Virtualized Logs Drawer Tab

**Files:**
- Create: `internal/ui/views/inspector/ringbuffer.go`
- Create: `internal/ui/views/inspector/tab_logs.go`
- Create: `internal/ui/views/inspector/tab_logs_test.go`

**Interfaces:**
- Consumes: Stream of `engine.LogEntry`, search query, follow flag, previous-run banner flag.
- Produces:
  ```go
  type LogBuffer struct {
      // Ring buffer capped at 10,000 entries
      Append(entry engine.LogEntry)
      Lines() []engine.LogEntry
      Len() int
      Clear()
  }

  type LogsViewModel struct {
      Buffer       *LogBuffer
      Follow       bool
      UnreadCount  int
      SearchQuery  string
      LevelFilter  string // "all", "error", "warn"
      IsPrevious   bool
      DriverErr    error
  }

  func (m *LogsViewModel) Render(s style.Sheet, w, h int) string
  ```

- [ ] **Step 1: Write failing tests for ring buffer and log view**
Write `internal/ui/views/inspector/tab_logs_test.go` testing:
- Bounded capacity: appending 12,000 lines retains exactly 10,000 lines (oldest 2,000 evicted).
- Paused follow: appending while `Follow == false` increments `UnreadCount` and shows badge `[↓ N new lines]`.
- Resuming follow (`Follow = true`) resets `UnreadCount = 0`.
- Previous logs mode: renders amber warning banner `[Viewing logs prior to restart at <time> · Press 'l' to resume live tail]`.
- Review Focus 2: When `DriverErr == ErrLogsNotSupported`, renders an informative explanation without crashing.
- Search filter: filters lines matching query string case-insensitively.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ui/views/inspector -run TestLogs`
Expected: FAIL.

- [ ] **Step 3: Implement `ringbuffer.go` and `tab_logs.go`**
Implement the ring buffer with 10,000 capacity and the log view with ANSI parsing, auto-scroll management, search filtering, and driver error display.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ui/views/inspector -run TestLogs`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/views/inspector/ringbuffer.go internal/ui/views/inspector/tab_logs.go internal/ui/views/inspector/tab_logs_test.go
git commit -m "feat(inspector): add bounded 10k virtualized logs viewer with follow toggle and search"
```

---

### Task 7: Dedicated Metrics & Sparklines Drawer Tab

**Files:**
- Create: `internal/ui/views/inspector/tab_metrics.go`
- Create: `internal/ui/views/inspector/tab_metrics_test.go`

**Interfaces:**
- Consumes: `engine.ContainerStats`, history `[]float64`, memory limit.
- Produces:
  ```go
  func RenderMetricsTab(s style.Sheet, stats engine.ContainerStats, cpuHistory []float64, isAlive bool, w, h int) string
  ```

- [ ] **Step 1: Write failing tests for metrics display and graceful degradation**
Write `internal/ui/views/inspector/tab_metrics_test.go` testing:
- Review Focus 3: When `isAlive == false` and no stats recorded, renders `Unavailable (container exited prior to session)`.
- When container has no memory limit (`MemoryLimit == 0`), renders `No memory limit configured` instead of 0% or NaN.
- Sparkline renderer draws ASCII/Unicode sparkline when history has points; renders `Collecting initial samples...` when empty.
- Memory threshold styling: Green (<70%), Yellow (70-90%), Red (>90%).

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ui/views/inspector -run TestMetrics`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/ui/views/inspector/tab_metrics.go`**
Implement the gauges, sparkline graph, threshold coloring, and graceful fallbacks for dead or unmeasured containers.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ui/views/inspector -run TestMetrics`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/views/inspector/tab_metrics.go internal/ui/views/inspector/tab_metrics_test.go
git commit -m "feat(inspector): add dedicated metrics tab with sparklines and graceful degradation"
```

---

### Task 8: Sensitive Env Masking & Safe Ports Drawer Tab

**Files:**
- Create: `internal/ui/views/inspector/tab_env.go`
- Create: `internal/ui/views/inspector/tab_env_test.go`

**Interfaces:**
- Consumes: `env []string`, `ports []string`, reveal state, timeout timer.
- Produces:
  ```go
  type EnvViewModel struct {
      EnvItems       []EnvItem
      RevealedAll    bool
      LastActivity   time.Time
  }

  type EnvItem struct {
      Key         string
      Value       string
      IsSensitive bool
      Revealed    bool
  }

  func ParseEnv(raw []string) []EnvItem
  func IsSensitiveKey(key string) bool
  func (m *EnvViewModel) CheckAutoRemask(now time.Time) bool
  func RenderEnvTab(s style.Sheet, m *EnvViewModel, ports []string, w, h int) string
  ```

- [ ] **Step 1: Write failing tests for secret masking and timeout**
Write `internal/ui/views/inspector/tab_env_test.go` testing:
- Review Focus 5: Case-insensitive regex matching identifies `AWS_SECRET_ACCESS_KEY`, `db_password`, `TOKEN`, `auth_bearer`, `Api_Key`, `PRIVATE_CERT`.
- Masked values render as `••••••••••••`.
- Pressing reveal on an item sets `Revealed = true`.
- `CheckAutoRemask` returns true and masks all values when `now.Sub(LastActivity) >= 60*time.Second`.
- Port mappings display target IP/port and security warning for non-loopback addresses (`0.0.0.0`).

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ui/views/inspector -run TestEnv`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/ui/views/inspector/tab_env.go`**
Implement secret detection regex, individual/bulk reveal logic, 60-second auto-remasking, and safe port confirmation presentation.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ui/views/inspector -run TestEnv`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/views/inspector/tab_env.go internal/ui/views/inspector/tab_env_test.go
git commit -m "feat(inspector): add sensitive env masking with 60s auto-remask and safe port launcher"
```

---

### Task 9: Chronological Timeline & Incident Sequence Drawer Tab

**Files:**
- Create: `internal/ui/views/inspector/tab_timeline.go`
- Create: `internal/ui/views/inspector/tab_timeline_test.go`

**Interfaces:**
- Consumes: Container inspect lifecycle dates (`Created`, `StartedAt`, `FinishedAt`), `HealthCheck.Log`, and Docker daemon events.
- Produces:
  ```go
  type TimelineItem struct {
      Timestamp time.Time
      Type      string // "create", "start", "health_warn", "oom", "die", "restart"
      Summary   string
      Outcome   string
  }

  func BuildIncidentSequence(c engine.ContainerInspection, events []engine.Event) ([]TimelineItem, bool)
  func RenderTimelineTab(s style.Sheet, items []TimelineItem, hasTruncatedHistory bool, w, h int) string
  ```

- [ ] **Step 1: Write failing tests for timeline incident sequence**
Write `internal/ui/views/inspector/tab_timeline_test.go` testing:
- Events are ordered chronologically: `Created -> Started -> Health Failing -> OOM -> Restart`.
- When Docker daemon events do not cover container creation time, `hasTruncatedHistory` is true.
- When `hasTruncatedHistory == true`, renders the integrity notice: `Earlier events prior to <time> are no longer retained by the Docker daemon.`

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ui/views/inspector -run TestTimeline`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/ui/views/inspector/tab_timeline.go`**
Implement event sorting, event type glyphs, relative time formatters, and integrity banner.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ui/views/inspector -run TestTimeline`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/views/inspector/tab_timeline.go internal/ui/views/inspector/tab_timeline_test.go
git commit -m "feat(inspector): add chronological incident sequence timeline with integrity notice"
```

---

### Task 10: Compose Project Grouping & Destructive Action Protections

**Files:**
- Create: `internal/ui/views/containers/compose_groups.go`
- Create: `internal/ui/components/confirm.go`
- Create: `internal/ui/components/confirm_test.go`
- Modify: `internal/app/actions.go`

**Interfaces:**
- Consumes: `[]engine.Container`, user action requests (`prune`, `remove`, `stack_stop`).
- Produces:
  ```go
  type ComposeGroup struct {
      Project   string
      Services  []engine.Container
      Collapsed bool
  }

  func GroupByCompose(containers []engine.Container) []ComposeGroup

  type ConfirmationDialog struct {
      Title       string
      Message     string
      TargetName  string
      Destructive bool
      Confirmed   func() tea.Cmd
  }
  ```

- [ ] **Step 1: Write failing tests for compose grouping and confirmation modals**
Write `internal/ui/components/confirm_test.go` testing:
- Grouping: containers with `Labels["com.docker.compose.project"]` group together under that project name; standalone containers appear in default group.
- Stopping a Compose stack does not set volume removal flags.
- Confirmation dialog for Volume Remove explicitly names the volume and warns about data deletion.
- Confirmation dialog for Image Prune displays count and reclaimable bytes before executing.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ui/components -run TestConfirm`
Expected: FAIL.

- [ ] **Step 3: Implement compose grouping and confirmation modals**
Implement `internal/ui/views/containers/compose_groups.go` and `internal/ui/components/confirm.go`, wiring into `internal/app/actions.go`.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ui/components -run TestConfirm`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ui/views/containers/compose_groups.go internal/ui/components/confirm.go internal/ui/components/confirm_test.go internal/app/actions.go
git commit -m "feat(containers): add compose project grouping and protected destructive confirmation modals"
```

---

### Task 11: End-to-End Master-Detail Integration & Verification

**Files:**
- Modify: `internal/app/model.go`
- Modify: `internal/app/view.go`
- Modify: `internal/app/input.go`
- Create: `internal/app/integration_test.go`

**Interfaces:**
- Consumes: All components from Tasks 1–10.
- Produces: Complete Lenger v0.2.0 master-detail application.

- [ ] **Step 1: Write comprehensive integration acceptance tests**
Write `internal/app/integration_test.go` asserting:
- App launch: 100% unobstructed list width.
- `Enter` on container: drawer opens at 35/65 ratio, latched to correct initial tab (Health for crashed, Logs for running).
- Moving `↑`/`↓` updates drawer container without closing or resetting tab.
- Responsive switch: terminal resize to 80 columns switches to overlay mode; resize to 120 columns restores split mode.
- `Tab` navigates between List $\leftrightarrow$ Drawer Tab Bar $\leftrightarrow$ Drawer Controls.
- `Esc` closes drawer and restores 100% list; cancels background log streams.
- Switching to `2: Images` and back to `1: Containers` restores container cursor and filter query.
- Full verification of Review Focus items 1–5.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/app -run TestMasterDetailIntegration`
Expected: FAIL.

- [ ] **Step 3: Wire components in `model.go`, `view.go`, and `input.go`**
Wire `ComputeLayout`, `FocusManager`, `RenderHealthTab`, `LogsViewModel`, `RenderMetricsTab`, `RenderEnvTab`, and `RenderTimelineTab` into Bubble Tea model, view, and update handlers. Ensure clean goroutine cancellation on drawer close.

- [ ] **Step 4: Run full test suite with race detector**
Run: `go test -race ./...`
Expected: ALL PASS with 0 race warnings.

- [ ] **Step 5: Build local binary and verify live**
Run:
```bash
go build -o /tmp/lenger-v2 ./cmd/lenger
LENGER_ENGINE=fake /tmp/lenger-v2 --version
```
Expected: PASS.

- [ ] **Step 6: Commit**
```bash
git add internal/app/model.go internal/app/view.go internal/app/input.go internal/app/integration_test.go
git commit -m "feat(app): complete Lenger v0.2.0 master-detail cockpit integration and verification"
```
