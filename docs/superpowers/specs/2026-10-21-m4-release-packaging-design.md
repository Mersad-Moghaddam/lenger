# Design Specification: Milestone M4 — Release, Packaging, Omarchy Integration & Performance

**Author:** Lenger Architecture Team  
**Date:** 2026-10-21  
**Status:** Approved  
**Target Milestone:** M4 (PRD.md Section 11.1)

---

## 1. Intent & Context

Milestone M4 prepares Lenger for general distribution and v1.0 release:
1. **Arch Linux & AUR Packaging (PRD 10.3, 11.1):** As Omarchy's base is Arch Linux, primary distribution requires clean `PKGBUILD` for source builds (`lenger`) and prebuilt binary builds (`lenger-bin`), plus a freedesktop `.desktop` launcher.
2. **Multi-Arch Releases (PRD 8):** GoReleaser configuration generating static CGO-free binaries for Linux (`amd64`, `arm64`, `armv7`) and macOS (`amd64`, `arm64`).
3. **Omarchy Desktop Setup (`lenger setup-omarchy`, PRD 6.6, 7.3):** Prints or interactively appends a non-conflicting keybinding (`Super+Alt+D`, avoiding lazydocker's `Super+Shift+D`) to `~/.config/hypr/bindings.conf` with automatic detection of Omarchy terminals (foot, kitty, alacritty, ghostty).
4. **Rich Documentation & VHS Tapes (PRD 11.2):** Polished README with instrument aesthetic screenshots/tapes, feature matrix, installation guides, and complete VHS tape recordings for CI artifact generation.
5. **Performance Budget Pass (PRD 8.2, 11.1):** Validates the 5,000-image benchmark fixture: store filtering in <10ms, memory overhead <30MB, UI list window rendering in <16ms without unbounded slice allocations.

---

## 2. Architecture & File Layout

### 2.1 Packaging (`packaging/`)
- `packaging/aur/PKGBUILD`: Source package building with `go build -trimpath -ldflags="-s -w -X main.version=$pkgver"`.
- `packaging/aur/lenger.install`: Post-install message suggesting `lenger setup-omarchy`.
- `packaging/aur-bin/PKGBUILD`: Prebuilt binary package downloading GitHub release archive.
- `packaging/lenger.desktop`: XDG desktop entry launching Lenger in default terminal.
- `.goreleaser.yaml`: GoReleaser specification producing static binaries, checksums, and changelog.

### 2.2 Omarchy Setup (`internal/platform/omarchy/setup.go`)
- Inspects `~/.config/hypr/bindings.conf` and `~/.config/omarchy/`.
- Detects whether `Super+Alt+D` or `Super+Shift+D` is already bound.
- Checks which terminal is installed: `ghostty`, `foot`, `kitty`, `alacritty`.
- Formats exact Hyprland `bind` directive:
  `bind = $mainMod ALT, D, exec, $terminal -e lenger`
- Safe append mode: prints diff, prompts for confirmation, never overwrites existing lines.

### 2.3 Performance Test Suite (`internal/core/store/perf_test.go`, `internal/ui/views/perf_test.go`)
- Generates 5,000 synthetic Docker image objects.
- Benchmarks:
  - `Store.Replace` and `Store.Images` retrieval.
  - Sorting 5,000 images by name, size, age.
  - Substring and prefix filtering on 5,000 images.
  - View window rendering (`ImageList` window of 30 lines) allocating 0 heap allocs per frame after window slicing.

### 2.4 VHS Tapes (`docs/vhs/`)
- `docs/vhs/demo.tape`: Complete walk-through recording.
- `docs/vhs/registry.tape`: Registry search, remote tag enumeration, and update checking.
- `docs/vhs/cleanup.tape`: Smart cleanup preview and volume confirmation.

---

## 3. Acceptance Criteria
1. `PKGBUILD` and `lenger.desktop` files are valid and adhere to Arch Packaging Standards.
2. `lenger setup-omarchy` outputs clear, safe Hyprland bindings without modifying user configuration without explicit confirmation.
3. Performance benchmarks verify that 5,000 images filter in under 10ms and memory remains under the 30MB budget.
4. Comprehensive README.md documents all Lenger features, keybindings, and Omarchy integration.
5. VHS tapes capture reproducible terminal recordings.
