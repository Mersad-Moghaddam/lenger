# Milestone M4: Release, Packaging, Omarchy Integration & Performance Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Milestone M4 from PRD.md: Packaging (AUR source, AUR bin, Desktop entry, GoReleaser config), Omarchy Setup command (`lenger setup-omarchy`), comprehensive Documentation and VHS tape scripts, and 5,000-image performance validation suite.

**Architecture:** Packaging specifications in `packaging/` and `.goreleaser.yaml`; desktop keybinding & terminal detection in `internal/platform/omarchy/setup.go`; performance benchmark suite testing store and view memory bounds under heavy loads; complete documentation in `README.md`.

**Tech Stack:** Go 1.27, Arch Linux PKGBUILD, GoReleaser, Bubble Tea v2, VHS.

**Spec:** `docs/superpowers/specs/2026-10-21-m4-release-packaging-design.md`

## Global Constraints

- Never mutate user Hyprland or Omarchy configuration files without explicit user approval.
- Follow Arch Packaging Guidelines strictly for `PKGBUILD` variables, permissions, and dependencies.
- Static CGO-free binaries for all release targets.
- 5,000-image load test must demonstrate sub-16ms render times and under 30MB memory footprint.
- User instruction: "dont build test or heavy run - just foucs on coding" — format with Docker gofmt, review diffs, and commit each phase.

## Review Focus

- PKGBUILD correctly sets architecture, license (MIT), dependencies, and build flags (`-trimpath -ldflags="-s -w"`).
- Desktop entry validates against standard Freedesktop specifications (`Terminal=true`, `Categories=Utility;System;`).
- Omarchy setup command detects non-conflicting keybindings (avoiding `Super+Shift+D` which lazydocker uses).
- GoReleaser YAML generates static multi-arch archives (`linux/amd64`, `linux/arm64`, `darwin/arm64`).
- Benchmark test verifies that store filtering and image list windowing scale smoothly to 5,000 images without allocating full-list copies.

---

### Task 1: Arch Linux & AUR Packaging

**Files:**
- Create: `packaging/aur/PKGBUILD`
- Create: `packaging/aur/lenger.install`
- Create: `packaging/aur-bin/PKGBUILD`
- Create: `packaging/lenger.desktop`
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Write `packaging/aur/PKGBUILD` and `lenger.install` for source builds**
- [ ] **Step 2: Write `packaging/aur-bin/PKGBUILD` for prebuilt binary releases**
- [ ] **Step 3: Write `packaging/lenger.desktop` Freedesktop launcher**
- [ ] **Step 4: Write `.goreleaser.yaml` for automated multi-architecture releases**

---

### Task 2: Omarchy Setup Integration

**Files:**
- Create: `internal/platform/omarchy/setup.go`
- Create: `internal/platform/omarchy/setup_test.go`
- Modify: `cmd/lenger/main.go`

- [ ] **Step 1: Implement terminal detection and keybinding generator in `internal/platform/omarchy/setup.go`**
- [ ] **Step 2: Add unit tests for setup helper in `internal/platform/omarchy/setup_test.go`**
- [ ] **Step 3: Connect command flag `--apply` in `cmd/lenger/main.go`**
- [ ] **Step 4: Format with Docker gofmt**

---

### Task 3: Performance Benchmark Fixtures & Budget Pass

**Files:**
- Create: `internal/core/store/perf_test.go`
- Create: `internal/ui/views/perf_test.go`

- [ ] **Step 1: Implement 5,000-image store benchmark in `internal/core/store/perf_test.go`**
- [ ] **Step 2: Implement 5,000-image UI view windowing benchmark in `internal/ui/views/perf_test.go`**
- [ ] **Step 3: Format with Docker gofmt**

---

### Task 4: Documentation & VHS Recordings

**Files:**
- Modify: `README.md`
- Create: `docs/vhs/demo.tape`
- Create: `docs/vhs/registry.tape`
- Create: `docs/vhs/cleanup.tape`

- [ ] **Step 1: Create VHS tapes for terminal recordings in `docs/vhs/`**
- [ ] **Step 2: Write comprehensive, polished `README.md` with features, shortcuts, and architecture**

---

### Task 5: Final Review, Formatting, Commit & Push

**Files:**
- Review git diff across repository
- Run Docker gofmt

- [ ] **Step 1: Run Docker gofmt across the repository**
- [ ] **Step 2: Review all staged changes, commit Milestone M4, and push to `origin/main`**
