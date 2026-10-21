# M2: Projects, Networks, System Info, and Dashboard Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver PRD Milestone M2: Compose project auto-discovery & lifecycle actions (PRJ-01–PRJ-05), Network list/inspection/lifecycle (NET-01), Daemon info & live event feed (SYS-03, SYS-04), Container healthcheck probe details (CTR-07), and refined Dashboard bento grid.

**Architecture:** Extend engine abstraction with Network and DaemonInfo interfaces and Compose runner; update fake engine and Docker Moby client adapter; extend normalized store with projects and networks; introduce project and network services; replace view stubs with responsive harbor-instrument views; wire keybindings, overlays, and offline gallery frames.

**Tech Stack:** Go 1.27, Bubble Tea v2, Lip Gloss v2, Moby Client v0.6.2, Harmonica, `chart` package.

**Spec:** `docs/PRD.md` (Sections 6.3, 6.4, 7.5, 9.2, 11.1 M2).

## Global Constraints

- Never import `internal/engine/docker` into UI or view packages; communicate solely through `services.Service` and `store.Store`.
- Update loops must never block: all I/O, compose commands, and network operations execute in `tea.Cmd`.
- No hardcoded colors or ad-hoc padding outside tokens.
- All views must degrade cleanly from 200×60 down to 80×24.
- Destructive actions (compose `down -v`, network removal) require explicit confirmation dialogs.
- Respect user constraint: coding, review, and formatting only; no build/test or heavy test suite executions.

## Review Focus

1. Containers without Compose labels must not panic project grouping or cause duplicate nil-project entries.
2. Compose down with `-v` must strictly require explicit modal toggle or typed confirmation to protect persistent volumes.
3. Network disconnection must guard against disconnecting the active Docker daemon connection or invalid endpoint references.
4. Daemon info retrieval failure must gracefully degrade to partial telemetry without disabling the System tab or disk usage.
5. Container inspection without healthcheck config must leave `Healthcheck` probe clean without nil dereference.

---

### Task 1: Engine Data Types and Interfaces for M2

**Files:**
- Modify: `internal/engine/engine.go`
- Modify: `internal/engine/errors.go`

**Interfaces:**
- Produces:
  - `type Network struct { ID, Name, Driver, Scope string; Internal, Attachable, Ingress, IPv6 bool; Subnets, Gateways []string; Containers []NetworkEndpoint; Labels map[string]string; Created time.Time }`
  - `type NetworkEndpoint struct { ContainerID, Name, IPv4, IPv6, MacAddress string }`
  - `type DaemonInfo struct { ID, Name, ServerVersion, APIVersion, MinAPIVersion, OS, OSType, Arch, KernelVersion, Driver, LoggingDriver, CgroupDriver, CgroupVersion, RootDir string; NCPU int; MemTotal int64; SecurityOptions, Warnings []string }`
  - `type ComposeProject struct { Name, WorkingDir string; ConfigFiles []string; Services []string; Status string; Containers []Container }`
  - `type ComposeAction string` (`ComposeUp`, `ComposeDown`, `ComposeRestart`, `ComposePull`)
  - `type ComposeOptions struct { ProjectDir string; ConfigFiles []string; ProjectName string; RemoveVolumes bool; Services []string }`
  - `type HealthProbe struct { Status string; FailingStreak int; Start, End time.Time; ExitCode int; Output string }`
  - Added to `ContainerInspection`: `HealthCheck *HealthProbe`
  - Added to `Engine`:
    - `ListNetworks(context.Context) ([]Network, error)`
    - `InspectNetwork(context.Context, string) (Network, error)`
    - `CreateNetwork(context.Context, string, string) (string, error)`
    - `RemoveNetwork(context.Context, string) error`
    - `ConnectNetwork(context.Context, string, string) error`
    - `DisconnectNetwork(context.Context, string, string, bool) error`
    - `DaemonInfo(context.Context) (DaemonInfo, error)`
    - `Compose(context.Context, ComposeAction, ComposeOptions, func(string) error) error`

- [ ] **Step 1: Declare M2 types and methods in `internal/engine/engine.go`**
- [ ] **Step 2: Add typed Compose and Network error helpers in `internal/engine/errors.go`**
- [ ] **Step 3: Verify formatting and syntax with Docker gofmt**

---

### Task 2: Engine Docker Adapter for Networks, Daemon Info, and Compose

**Files:**
- Create: `internal/engine/docker/network.go`
- Create: `internal/engine/docker/info.go`
- Create: `internal/engine/docker/compose.go`
- Modify: `internal/engine/docker/container.go`

**Interfaces:**
- Consumes: `client.Client` from `github.com/moby/moby/client`
- Produces: Complete `engine.Engine` implementation for networks, system info, compose runner, and health probe extraction.

- [ ] **Step 1: Implement `internal/engine/docker/network.go` for network listing, inspect, create, remove, connect, disconnect**
- [ ] **Step 2: Implement `internal/engine/docker/info.go` reading `client.Info` and `client.ServerVersion` into `engine.DaemonInfo`**
- [ ] **Step 3: Implement `internal/engine/docker/compose.go` running `docker compose` with streaming JSON/log output**
- [ ] **Step 4: Update `InspectContainer` in `internal/engine/docker/container.go` to extract `c.State.Health.Log` into `HealthProbe`**
- [ ] **Step 5: Verify formatting with Docker gofmt**

---

### Task 3: Engine Fake Adapter for Networks, Daemon Info, and Compose

**Files:**
- Create: `internal/engine/fake/network.go`
- Create: `internal/engine/fake/info.go`
- Create: `internal/engine/fake/compose.go`
- Modify: `internal/engine/fake/fake.go`
- Modify: `internal/engine/fake/container.go`

**Interfaces:**
- Produces: Deterministic offline responses for networks (`bridge`, `storefront_default`), daemon info (Docker 27.3.1, 16 CPUs, 32GB RAM), compose execution, and container health probes.

- [ ] **Step 1: Implement `internal/engine/fake/network.go` with fixture networks and thread-safe connect/disconnect/create/remove**
- [ ] **Step 2: Implement `internal/engine/fake/info.go` with deterministic daemon details**
- [ ] **Step 3: Implement `internal/engine/fake/compose.go` with scripted compose up/down log stream**
- [ ] **Step 4: Update `InspectContainer` in `internal/engine/fake/container.go` with realistic health check results**
- [ ] **Step 5: Verify formatting with Docker gofmt**

---

### Task 4: Store Normalization & Core Services for Projects and Networks

**Files:**
- Modify: `internal/core/store/store.go`
- Create: `internal/core/services/projects.go`
- Create: `internal/core/services/networks.go`
- Modify: `internal/core/services/services.go`

**Interfaces:**
- Produces:
  - `Store.Networks []engine.Network`
  - `Store.Projects []engine.ComposeProject`
  - `Store.Daemon engine.DaemonInfo`
  - `Store.ByProject map[string][]engine.Container`
  - `Store.ByNetwork map[string][]engine.Container`
  - `Service.ListProjects() []engine.ComposeProject`
  - `Service.Compose(ctx, action, options, emit) error`
  - `Service.ListNetworks(ctx) ([]engine.Network, error)`
  - `Service.CreateNetwork(ctx, name, driver) (string, error)`
  - `Service.RemoveNetwork(ctx, id) error`
  - `Service.ConnectNetwork(ctx, netID, containerID) error`
  - `Service.DisconnectNetwork(ctx, netID, containerID, force) error`
  - `Service.DaemonInfo(ctx) (engine.DaemonInfo, error)`

- [ ] **Step 1: Update `internal/core/store/store.go` to group containers into projects and networks**
- [ ] **Step 2: Implement `internal/core/services/projects.go` for compose actions and project discovery**
- [ ] **Step 3: Implement `internal/core/services/networks.go` for network validation and operations**
- [ ] **Step 4: Update `internal/core/services/services.go` with timeout policies for compose and networks**
- [ ] **Step 5: Verify formatting with Docker gofmt**

---

### Task 5: UI Views for Projects, Networks, and Enhanced System & Containers

**Files:**
- Create: `internal/ui/views/projects.go`
- Create: `internal/ui/views/networks.go`
- Modify: `internal/ui/views/system.go`
- Modify: `internal/ui/views/containers.go`
- Modify: `internal/ui/views/dashboard.go`

**Interfaces:**
- Produces:
  - `views.Projects(s style.Sheet, items []engine.ComposeProject, cursor int, filter string, streamOutput []string, width, height int, now time.Time) string`
  - `views.Networks(s style.Sheet, items []engine.Network, cursor int, filter string, width, height int, now time.Time) string`
  - Updated `views.System` rendering the Daemon Info panel alongside disk usage and cleanup
  - Updated `views.ContainerDetail` showing the CTR-07 Health Probe block (exit code, output, streak, last probe)
  - Updated `views.Dashboard` bento grid with real event stream, health counts, and storage soundings

- [ ] **Step 1: Implement `internal/ui/views/projects.go` with service breakdown and streaming log split pane**
- [ ] **Step 2: Implement `internal/ui/views/networks.go` with subnets, IPAM, and attached endpoint list**
- [ ] **Step 3: Enhance `internal/ui/views/system.go` with DAEMON INFORMATION panel**
- [ ] **Step 4: Enhance `internal/ui/views/containers.go` with HEALTHCHECK PROBE details in ContainerDetail**
- [ ] **Step 5: Enhance `internal/ui/views/dashboard.go` bento grid with live health and events**
- [ ] **Step 6: Verify formatting with Docker gofmt**

---

### Task 6: App Model Routing, Workflows, Overlays, and Keybindings for M2

**Files:**
- Modify: `internal/app/model.go`
- Modify: `internal/app/view.go`
- Modify: `internal/app/input.go`
- Modify: `internal/app/workflows.go`
- Modify: `internal/app/overlays.go`
- Modify: `internal/gallery/gallery.go`

**Interfaces:**
- Integrates:
  - Tab 3 (Projects): Navigation, `u` (compose up), `d` (compose down), `r` (restart), `l` (project logs), `/` (filter)
  - Tab 5 (Networks): Navigation, `d` (remove network), `c` (create network), `/` (filter)
  - Compose action confirmation overlay with optional volume removal (`-v`) toggle
  - Network creation overlay and network removal confirmation
  - Offline gallery frames for `projects`, `networks`, `compose-action`, and `create-network`

- [ ] **Step 1: Add project, network, and compose state fields to `Model` in `internal/app/model.go`**
- [ ] **Step 2: Route views in `internal/app/view.go` for tab 3 (`Projects`) and tab 5 (`Networks`)**
- [ ] **Step 3: Implement keybindings and filter overlays in `internal/app/input.go` for tabs 3 and 5**
- [ ] **Step 4: Implement compose and network async commands and workers in `internal/app/workflows.go`**
- [ ] **Step 5: Implement modal overlays for compose action and network creation in `internal/app/overlays.go`**
- [ ] **Step 6: Seed gallery frames in `internal/gallery/gallery.go` for projects and networks**
- [ ] **Step 7: Verify formatting with Docker gofmt**

---

### Task 7: Unit Tests & Verification for M2

**Files:**
- Create: `internal/ui/views/m2_test.go`
- Create: `internal/app/m2_test.go`

**Interfaces:**
- Tests:
  - Compose project grouping by labels
  - Compose action routing and typed `-v` protection
  - Network creation, disconnect, and removal routing
  - Container health probe display formatting
  - Daemon info formatting in system view

- [ ] **Step 1: Write `internal/ui/views/m2_test.go` for Projects, Networks, and Health probe rendering**
- [ ] **Step 2: Write `internal/app/m2_test.go` for M2 workflow routing and message handling**
- [ ] **Step 3: Run code formatting, git diff check, and review all additions**
- [ ] **Step 4: Commit and push M2 milestone**
