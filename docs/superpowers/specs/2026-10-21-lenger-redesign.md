# Lenger Redesign Specification: Master-Detail TUI & Diagnostic Cockpit

**Date:** 2026-10-21  
**Target Release:** v0.2.0  
**Status:** Draft for Builder Review  
**Package:** `github.com/mersad/lenger`

---

## 1. Intent & The Why

Existing terminal Docker dashboards (such as LazyDocker) suffer from severe multi-pane claustrophobia: dividing the terminal into 4–5 small fixed boxes (Containers, Images, Volumes, Networks, Logs) where each box only shows 2–3 rows. Developers spend an excessive amount of time tabbing between tiny boxes, reading truncated text, and struggling to understand *why* a service crashed or restart-looped.

**Lenger** is redesigned from the ground up to replace multi-pane friction with a calm, progressive disclosure experience:
1. **Calm Overview:** A spacious, single-column container list providing immediate visibility into container health, resource utilization, and native status without noise.
2. **Stable Master-Detail Inspection:** Opening an inspector drawer never disrupts the list's cursor, scroll position, or sorting.
3. **Evidence-Grounded Diagnostics:** Plain-English explanations of failures that rigorously separate verified Docker daemon facts from resource telemetry and diagnostic hypotheses.
4. **Security & Operability by Default:** Masked secrets, safe port launching, bounded virtualized logs, and accessible standard keyboard navigation.

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
- **Stable Master-Detail Cockpit for Containers:** A rebalanced 35% list / 65% inspector default split with a user-resizable divider, serving as the default primary view (`1: Containers`).
- **Retained Lean Resource Views:** Maintain clean, single-table views for other essential Docker entities (`2: Images`, `3: Volumes`, `4: Networks`) without multi-pane clutter.
- **Integrated Compose Project Grouping:** Compose projects group containers naturally inside the Containers view with project-level batch actions (e.g., restart stack).
- **Context Preservation Across Navigation:** Switching to Images/Volumes/Networks suspends the container drawer and pauses active background streams; returning to Containers immediately restores the previous selection, scroll offset, and filter state.
- **Native Docker State Model:** Strict adherence to Docker Engine lifecycle terms (`Exited (137)`, `Restarting · N attempts`, `Unhealthy · running`, `Running (healthy)`).
- **Traceable Diagnostic Provenance:** Clear separation of verified Docker inspect facts, telemetry measurements, and probable cause hypotheses.
- **Dedicated 5-Tab Drawer:**
  - `Health & Diagnostics`
  - `Live Logs`
  - `Metrics`
  - `Env & Ports`
  - `Timeline`
- **Predictable Tab Latching:** The initial tab is selected once when the drawer opens based on container health and is never yanked or auto-switched while the developer is inspecting.
- **Accessible Keyboard & Mouse Hierarchy:** Standard `Tab` / `Shift+Tab` panel focus, arrow navigation, explicit `Enter` activation, `Esc` dismissal, and `z` fullscreen zoom.
- **Security-First Environment Display:** Case-insensitive masking of sensitive tokens with automatic timeout and safe port opening previews.
- **Bounded Virtualized Logs:** 10,000-line memory ring buffer, independent background collection while auto-scroll is paused, and fast search.

### Non-Goals
- **Kubernetes Cluster Management:** Lenger is focused entirely on local and remote Docker Engine / Docker Compose daemon workflows.
- **Full Cloud Orchestration:** No swarm/cluster orchestration beyond standard Docker engine APIs.
- **Heavy Image/Registry Editing:** Advanced multi-stage layer manipulation remains the domain of specialized CLI tools.

### Anti-Goals
- **No Multi-Pane Fragmentation:** Never display 4+ simultaneous micro-panes that cramp text.
- **No Ambiguous Hypotheses as Facts:** Never claim a container died of OOM based solely on exit code 137 unless verified by `State.OOMKilled` or cgroup metrics.
- **No Background UI Hijacking:** Background container status changes must never switch tabs or pop up windows over an active debugging session.
- **No Silent Secret Leakage:** Revealed secrets must never be written to logs, diagnostic exports, or screen captures.
- **No Background Socket Waste:** Navigating away from Containers must pause all active streaming log and stats goroutines.

---

## 3. Layout & Master-Detail Architecture

### 3.1 Split Ratio & Breakpoint Behavior
- **Default Split:** 35% container list (left), 65% inspector drawer (right).
- **Drawer Lifecycle & Collapsed State:**
  - When Lenger launches or when the user closes the drawer (`Esc`), the drawer is **completely collapsed**, and the container list expands to **100% unobstructed terminal width**.
  - Arrow keys (`↑`/`↓`) navigate the list cleanly without opening the drawer.
  - Pressing `Enter` or clicking a container opens the drawer at the configured ratio (35/65 default).
  - Pressing `Esc` closes the drawer and restores the 100% unobstructed list immediately. Active streaming and stats goroutines for the drawer are suspended upon closure to conserve resources.
- **Divider Adjustment:** The divider position is adjustable via mouse drag or keyboard shortcut (`[` shrink list, `]` expand list) and persisted in user state (`~/.config/lenger/state.json`).
- **Minimum Usable Widths:**
  - Minimum List Width: 32 columns.
  - Minimum Inspector Width: 54 columns.
- **Responsive Mode Switch:**
  - When terminal width $\ge$ `(MinListWidth + MinInspectorWidth + 1)` (87 columns), opening the drawer renders in **Side-by-Side Split**.
  - When terminal width $<$ 87 columns, opening the drawer renders in **Slide-Over Overlay Mode** where the drawer renders full width over the list with an explicit `[Esc] Back` indicator.
  - On resize, Lenger maintains the selected container and active tab without state loss.

### 3.2 List Pane (Left, 35%)
- **Rows:** Each row displays:
  - Status dot: Green (Healthy/Running), Red (Exited $\ne$ 0 / Unhealthy / Restarting), Yellow (Starting / Paused), Gray (Exited 0).
  - Container name (bold, truncated with ellipsis if exceeding column budget).
  - Compose project tag (if container is part of a compose stack).
  - Quick resource/status text: CPU% or concise status string (`Exited (137)`).
- **Compose Project Grouping:** Containers belonging to a Docker Compose stack can be grouped under collapsible headers (`📦 <project-name>`), with project-level actions (`Restart Stack`, `Stop Stack`).
- **List Stability:**
  - Up/Down navigation moves the highlight cursor.
  - When the drawer is open, moving the cursor in the list updates the drawer contents in real time without closing or resetting the drawer tab.
  - Background container status updates refresh row indicators in place; they never sort or reorder rows while the user is navigating.

### 3.3 Top Navigation & Resource Views
Lenger organizes Docker resources via simple, discoverable top-level tabs:
- `1: Containers` (Default) — The high-efficiency Master-Detail Cockpit with 5-tab inspector drawer.
- `2: Images` — Lean, clean table: Repository, Tag, Image ID, Created, Virtual Size, and In-Use status. Actions: `p` prune dangling images, `d` remove image, `Enter` inspect image details/layers.
- `3: Volumes` — Lean, clean table: Driver, Volume Name, Mount Point, Associated Containers, Size on disk (when available). Actions: `p` prune unused volumes, `d` remove volume.
- `4: Networks` — Lean, clean table: Name, Driver, Scope, Subnet, Gateway, Connected Containers count. Actions: `d` remove custom network, `Enter` inspect subnet IP assignments.

**Context Preservation Rules:**
- Navigating between resource types (`1` through `4`) preserves the Containers view state: the search/filter query, the selected container ID, and the vertical scroll position.
- If the diagnostic drawer was open on `1: Containers` when switching to another resource, the drawer is cleanly suspended and all active streaming log readers and polling stats goroutines are canceled immediately to eliminate CPU and Docker socket overhead.
- Returning to `1: Containers` restores the previous list selection and scroll position. If the container still exists, the user can press `Enter` to resume inspection with fresh real-time data.

---

## 4. Inspector Drawer Tabs & Provenance Model

### 4.1 Tab Latching Logic
When the drawer is opened (via `Enter` or mouse click):
1. **Exited ($\ne$ 0), Restarting, Unhealthy, or Paused:** Latches to **Health & Diagnostics**.
2. **Running (Healthy or no health check):** Latches to **Live Logs**.
3. **Starting or Health Unknown:** Latches to **Health & Diagnostics** (with explicit message that health checks are initializing).

*Stability Invariant:* Once open, background state changes never switch the active tab. If an error or crash occurs while viewing Live Logs, an indicator badge appears on the Health tab header (`[Health (ERR)]`).

---

### 4.2 Tab 1: Health & Diagnostics
The diagnostics engine presents information in three strictly segregated tiers:

1. **Tier 1 — Verified Daemon Evidence:**
   - Source: Direct Docker Engine inspect payload (`types.ContainerJSON`).
   - Fields:
     - `State.Status` and `State.ExitCode`.
     - `State.OOMKilled` (`true` / `false`).
     - `State.Error` (daemon error string if present).
     - `RestartCount` and restart policy (`on-failure`, `always`, etc.).
     - `Config.Healthcheck` configuration and `State.Health.Log` (last N healthcheck outputs with exit codes and execution durations).
   - Provenance tag: `[Docker Engine Inspect · <timestamp>]`.

2. **Tier 2 — Resource Observations:**
   - Source: Sampled Docker stats via Docker Engine API (`ContainerStats`). Gathering is performed exclusively via Docker API (never direct host `/sys/fs/cgroup` files) to ensure uniform operation over local sockets, remote `DOCKER_HOST`, rootless Docker, and macOS/WSL2.
   - Fields: Peak memory observed vs configured memory limit (`HostConfig.Memory`).
   - Inactive/Exited Handling: If Lenger observed stats while the container was running earlier in the session, that recorded peak is displayed with a session timestamp. If the container exited before Lenger launched, peak usage is explicitly marked as `Unavailable (container exited prior to telemetry session)`.
   - Provenance tag: `[Telemetry: Docker API ContainerStats · Sampled at <time>]`.

3. **Tier 3 — Diagnostic Hypothesis & Next Actions:**
   - Clearly labeled as an inference (e.g., `Hypothesis: Container exceeded memory limit during peak traffic`).
   - **Action Buttons:**
     - `[p] View Previous Instance Logs`: Queries `ContainerLogs` with `Until = State.StartedAt` to fetch output from the run immediately preceding the last restart. This seamlessly transitions the view to Tab 2 (Logs) pinned in a historical viewing mode with an amber warning banner: `[Viewing logs prior to restart at <time> · Press 'l' to resume live tail]`.
     - `[r] Restart Container`.
     - `[c] View Failing Healthcheck Command`.

---

### 4.3 Tab 2: Live Logs
- **Streaming Pipeline:** Reads multiplexed stdout/stderr via Docker API `ContainerLogs`.
- **Buffer:** Bounded ring buffer holding up to 10,000 lines.
- **Coloring:** Full ANSI 256-color / 24-bit TrueColor escape sequence pass-through.
- **Auto-Scroll & Background Collection:**
  - When auto-scroll is active, new lines scroll into view automatically.
  - When the user scrolls up, auto-scroll pauses. **Log collection does NOT stop.** New incoming lines continue buffering in the background, and an unread counter appears: `[↓ 24 new lines]`.
  - Pressing `f` or `G` (End) jumps to bottom and resumes auto-scroll.
- **Search & Filtering:**
  - Pressing `/` opens in-drawer filter input to search text.
  - Level filter toggle: `[All]` / `[Errors]` / `[Warnings]`.
- **Copy/Export:** `y` copies visible/selected log lines to system clipboard.

---

### 4.4 Tab 3: Metrics
- **Current Resource Gauges:**
  - CPU Usage percentage with visual Lipgloss bar meter.
  - Memory Usage vs Configured Limit (e.g. `210 MiB / 1024 MiB (20.5%)`) with color thresholds (Green < 70%, Yellow 70–90%, Red > 90%).
  - Network I/O (Bytes In / Bytes Out).
  - Block I/O (Disk Read / Write).
- **Historical Sparklines:**
  - If Lenger has sampled metrics during the session, displays compact sparkline graphs over the last 60 seconds / 5 minutes.
  - If no prior history exists, displays: `Telemetry collection active since <time>; historical data before this session is not available.`

---

### 4.5 Tab 4: Env & Ports
- **Sensitive Environment Variable Masking:**
  - Case-insensitive regex matching for sensitive keys: `*(PASSWORD|SECRET|KEY|TOKEN|AUTH|CREDENTIAL|PRIVATE|API_KEY)*`.
  - Masked values display as `••••••••••••`.
  - Individual reveal: Pressing `Space` or clicking a masked row reveals the value.
  - Reveal All: Pressing `u` reveals all values.
  - **Auto-Remask Safeguard:** All revealed values automatically re-mask when:
    - The drawer is closed.
    - The user selects a different container.
    - 60 seconds elapse without interaction.
  - Revealed secrets are stripped/redacted from diagnostic exports and log files.
- **Port Mapping & Safe Launch:**
  - Lists published and exposed ports: `0.0.0.0:8080 -> 80/tcp`.
  - Selecting a port displays a confirmation dialog showing the target URL (`http://127.0.0.1:8080`) before opening in the system browser.
  - Non-loopback addresses (`0.0.0.0` or external IPs) display an explicit security note.

---

### 4.6 Tab 5: Timeline & Events
- **Chronological Incident Sequence:**
  - Integrates Docker daemon events (`container create`, `start`, `die`, `restart`, `oom`) with healthcheck transitions.
  - Each item shows timestamp, event type, and outcome.
- **Integrity Notice:**
  - If Docker daemon event history does not contain earlier records (e.g., daemon restart or container created long before Lenger opened), an explicit notice is displayed: `Earlier events prior to <timestamp> are no longer retained by the Docker daemon.`

---

## 5. Keyboard Navigation & Interaction Hierarchy

### 5.1 Focus Management & Navigation Loop
Keyboard focus operates on a predictable two-pane hierarchy:
1. **List Focused (Default):**
   - `↑` / `↓` (`k` / `j`) navigates containers.
   - `Enter` opens the Inspector Drawer (latched to the smart initial tab).
   - If the drawer is already open, moving `↑` / `↓` in the list immediately updates the drawer for the newly highlighted container.
   - `Tab` moves focus into the **Inspector Drawer**.
2. **Drawer Focused:**
   - Focus begins on the **Tab Bar**: `←` / `→` (`h` / `l`) switches between the 5 tabs.
   - Pressing `Tab` moves focus into the active tab's interactive controls (e.g. action buttons in Tab 1, secret reveal rows in Tab 4, or search input).
   - `↑` / `↓` scrolls the drawer viewport (logs, inspect text, timeline).
   - `Tab` from the last control cycles focus back to the **Container List**.
   - `Shift+Tab` reverses the focus loop.
   - `Esc` closes the drawer and returns focus immediately to the Container List (or exits fullscreen if zoomed).
3. **Bottom Bar:**
   - The bottom bar acts as a static status and global keybind legend (not a focus stop). Global shortcuts (`r` restart, `s` stop/start, `z` zoom, `?` help, `q` quit) remain active regardless of whether the List or Drawer currently holds focus.

| Key | Context | Action |
|---|---|---|
| `1`–`4` | Global | Switch Resource View (`1` Containers, `2` Images, `3` Volumes, `4` Networks) |
| `Tab` | Global | Move focus: Container List &harr; Drawer Tab Bar &harr; Drawer Controls |
| `Shift+Tab` | Global | Reverse focus order |
| `↑` / `↓` (`k` / `j`) | List Focused | Navigate containers (updates drawer live if open) |
| `↑` / `↓` (`k` / `j`) | Drawer Focused | Scroll active drawer content (logs, config, timeline) |
| `←` / `→` (`h` / `l`) | Tab Bar Focused | Switch active drawer tab |
| `Enter` | List Focused | Open drawer on selected container |
| `Enter` | Control Focused | Activate focused button, open port preview, or reveal secret |
| `Esc` | Drawer Focused | Close drawer (or exit fullscreen if zoomed) and restore 100% list |
| `z` | Drawer Open | Toggle fullscreen zoom for the drawer |
| `[` / `]` | Split View | Resize master-detail divider (-5% / +5%) |
| `/` | Live Logs / List | Open search / filter prompt |
| `f` | Live Logs | Toggle follow / auto-scroll |
| `l` | Previous Logs | Return from previous-run logs to live tail |
| `r` | Global / Action | Restart selected container |
| `s` | Global / Action | Stop / Start toggle |
| `?` | Global | Open Keyboard Shortcuts & Help Modal |
| `q` / `Ctrl+C` | Global | Quit Lenger |

---

## 6. Technical Architecture & Component Structure

### 6.1 Package Decomposition
- `internal/app/`: Core Bubble Tea Model, global event routing, window size dispatch, overlay stack management.
- `internal/docker/`: Thread-safe Docker Engine client wrapper, streaming log readers, stats polling, inspect caching, event monitoring.
- `internal/ui/masterdetail/`: Master-detail layout engine, split ratio calculation, minimum width enforcement, resizable divider.
- `internal/ui/views/containers/`: Container list component with stable cursor navigation and compose grouping.
- `internal/ui/views/inspector/`: Five-tab inspector drawer component:
  - `tab_health.go`: Diagnostic engine, evidence provenance formatting, previous log fetching.
  - `tab_logs.go`: Bounded ring buffer, ANSI parser, search filter, auto-scroll manager.
  - `tab_metrics.go`: Real-time gauges, cgroup v2 stats, sparkline generators.
  - `tab_env.go`: Secret detection, reveal timeout timer, port confirmation modal.
  - `tab_timeline.go`: Event aggregator, timeline renderer, integrity checks.
- `internal/ui/tokens/`: Color palettes, border styles, status glyphs, and typography.

### 6.2 Concurrency & State Invariants
- **Thread Safety:** All Docker API calls (`ContainerLogs`, `ContainerStats`, `Events`) run in dedicated background goroutines and dispatch Bubble Tea `tea.Msg` to the main update loop.
- **Log Memory Cap:** Each active container log buffer uses a fixed-size ring buffer allocating no more than 10,000 lines (approx. 2–5 MB memory per active stream).
- **Cleanup:** Closing a drawer cancels active log and stats contexts immediately to avoid goroutine leaks.

---

## 7. What's Left to the Builder

The builder has full latitude to:
1. Choose specific Lipgloss styling details (padding, borders, color shades) consistent with `internal/ui/tokens`.
2. Implement the ring buffer data structure (`container/ring` or custom slice ring).
3. Determine internal sub-message types (`tea.Msg`) for asynchronous log chunks and event streaming.
4. Organize unit and integration test fixtures using the existing `testutil` framework.
