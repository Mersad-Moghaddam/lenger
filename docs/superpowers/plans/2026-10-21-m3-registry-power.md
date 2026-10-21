# Milestone M3: Registry Power & Image Mastery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Milestone M3 from PRD.md: Registry Search (Docker Hub/GHCR/Quay), Layer Breakdown & Biggest-Layer Sounding, Tagging/Pushing/Tar Export-Import, Remote Update Check with 1-Key Recreate, Pluggable Vulnerability Scanner Adapter (Trivy/Grype/Scout), Reverse Dependencies View ("Why is this image here?"), and Mirror-Aware Pulls with `/etc/docker/daemon.json` diff preview.

**Architecture:** Domain contracts extended in `internal/engine/`, implemented for Docker Moby client and Fake fixture engine; business logic for scanner CLI parsing, mirror rewriting, and update checking in `internal/core/services/`; enhanced Image views with layer charts, vulnerability status, and dependency trees in `internal/ui/views/`; interactive overlays and keybindings in `internal/app/`.

**Tech Stack:** Go 1.27, Bubble Tea v2, Lip Gloss v2, Moby Docker API client, distribution/reference.

**Spec:** `docs/superpowers/specs/2026-10-21-m3-registry-power-design.md`

## Global Constraints

- Never block the UI thread during registry network lookups or vulnerability scans; always use background commands with timeouts.
- Never store raw passwords; use Docker's credential helpers exclusively.
- All layer commands and inspection details must sanitize environment values and build secrets.
- Responsive down to 80×24 cells with ASCII fallback support.
- User instruction: "dont build test or heavy run - just foucs on coding" — format with Docker gofmt, write unit tests, and review code diffs before committing.

## Review Focus

- OCI registry manifest digest comparison handles both Schema 2 and OCI manifest index media types.
- Vulnerability scanner adapter gracefully handles when Trivy/Grype/Scout is not installed in PATH without errors.
- Mirror rewriting correctly handles library images (`nginx` -> `library/nginx`) and custom registries.
- Image export/import validates file paths and permissions before spawning workers.
- Layer breakdown calculations handle zero-sized layers and metadata-only steps without division by zero.

---

### Task 1: Engine Contracts & Error Classification

**Files:**
- Modify: `internal/engine/engine.go`
- Modify: `internal/engine/errors.go`

**Interfaces:**
- Produces: `RegistrySearchResult`, `RemoteTag`, `ImageUpdateStatus`, `VulnerabilityReport`, `LayerDetail`, and engine methods: `SearchRegistry`, `ListRemoteTags`, `CheckImageUpdate`, `TagImage`, `PushImage`, `SaveImage`, `LoadImage`.

- [ ] **Step 1: Declare M3 data structures and methods in `internal/engine/engine.go`**
- [ ] **Step 2: Add registry and mirror error classifications in `internal/engine/errors.go`**
- [ ] **Step 3: Format with Docker gofmt**

---

### Task 2: Docker Client & Fake Engine Adapters

**Files:**
- Create: `internal/engine/docker/registry.go`
- Create: `internal/engine/docker/tag.go`
- Create: `internal/engine/fake/registry.go`
- Modify: `internal/engine/docker/docker.go`
- Modify: `internal/engine/fake/fake.go`

**Interfaces:**
- Implements: `SearchRegistry`, `ListRemoteTags`, `CheckImageUpdate`, `TagImage`, `PushImage`, `SaveImage`, `LoadImage` across both engines.

- [ ] **Step 1: Implement Docker Moby client methods in `internal/engine/docker/registry.go` and `tag.go`**
- [ ] **Step 2: Implement deterministic fixture methods in `internal/engine/fake/registry.go` and `fake.go`**
- [ ] **Step 3: Format with Docker gofmt**

---

### Task 3: Core Services: Scanner, Mirrors & Registry

**Files:**
- Create: `internal/core/services/scanner.go`
- Create: `internal/core/services/mirrors.go`
- Create: `internal/core/services/registry.go`

**Interfaces:**
- Produces: `ScanVulnerabilities(ctx, imageID) (VulnerabilityReport, error)`
- Produces: `RewriteMirrorReference(ref, mirror string) (string, error)`
- Produces: `GenerateDaemonMirrorsDiff(existingJSON string, mirrors []string) (string, error)`
- Produces: `CheckUpdate(ctx, ref, localDigest string) (ImageUpdateStatus, error)`

- [ ] **Step 1: Implement scanner CLI detector and report parser in `internal/core/services/scanner.go`**
- [ ] **Step 2: Implement mirror reference rewriter and `/etc/docker/daemon.json` diff builder in `internal/core/services/mirrors.go`**
- [ ] **Step 3: Implement registry operations and update checks in `internal/core/services/registry.go`**
- [ ] **Step 4: Format with Docker gofmt**

---

### Task 4: UI Views: Layer Breakdown, Reverse Dependencies & Security Soundings

**Files:**
- Modify: `internal/ui/views/images.go`
- Create: `internal/ui/views/layers.go`
- Create: `internal/ui/views/dependencies.go`

**Interfaces:**
- Produces: `LayerSounding(layers []engine.Layer, width int, ascii bool) string`
- Produces: `ReverseDependencies(image engine.Image, containers []engine.Container, projects []engine.ComposeProject, width, height int) string`

- [ ] **Step 1: Implement layer size visualizer and biggest-layer callout in `internal/ui/views/layers.go`**
- [ ] **Step 2: Implement reverse dependency view ("Why is this image here?") in `internal/ui/views/dependencies.go`**
- [ ] **Step 3: Update `internal/ui/views/images.go` to integrate layer bar, vulnerability badges, and dependencies**
- [ ] **Step 4: Format with Docker gofmt**

---

### Task 5: Modal Overlays for Registry Search, Tagging, and Mirrors

**Files:**
- Modify: `internal/app/overlays.go`
- Create: `internal/app/registry_search.go`

**Interfaces:**
- Produces: `registrySearchOverlay()`, `tagImageOverlay()`, `daemonMirrorsOverlay()`, `exportImageOverlay()`

- [ ] **Step 1: Implement registry search and tag browser UI in `internal/app/registry_search.go`**
- [ ] **Step 2: Implement tag image and daemon mirrors diff overlays in `internal/app/overlays.go`**
- [ ] **Step 3: Format with Docker gofmt**

---

### Task 6: App State, Keybindings, Workflows & Gallery Integration

**Files:**
- Modify: `internal/app/model.go`
- Modify: `internal/app/view.go`
- Modify: `internal/app/input.go`
- Modify: `internal/app/workflows.go`
- Modify: `internal/gallery/gallery.go`

**Interfaces:**
- Adds keybindings: `S` (registry search), `U` (update check), `t` (tag image), `P` (push image), `E` (export tar)
- Wires async workers for scan, update check, search, and mirror fallback.
- Registers M3 screens in gallery.

- [ ] **Step 1: Update model fields and message routing in `internal/app/model.go`**
- [ ] **Step 2: Wire keybindings and actions in `internal/app/input.go` and view routing in `internal/app/view.go`**
- [ ] **Step 3: Implement background workflow executors in `internal/app/workflows.go`**
- [ ] **Step 4: Register M3 screens in `internal/gallery/gallery.go`**
- [ ] **Step 5: Format with Docker gofmt**

---

### Task 7: Unit Testing, Static Code Review & Milestone Verification

**Files:**
- Create: `internal/ui/views/m3_test.go`
- Create: `internal/app/m3_test.go`
- Create: `internal/core/services/m3_test.go`

**Interfaces:**
- Tests all M3 view components, services, and model interactions.

- [ ] **Step 1: Author unit test suites covering M3 functionality**
- [ ] **Step 2: Run Docker gofmt across all files**
- [ ] **Step 3: Review git diff, commit milestone M3, and push to `origin/main`**
