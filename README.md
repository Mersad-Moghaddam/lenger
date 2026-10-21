# Lenger (لنگر)

**Anchor your containers.** A fast, low-ceremony, harbor-instrument Docker TUI built for Linux with optional first-class Omarchy desktop integration.

[![Go Version](https://img.shields.io/badge/go-1.27+-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Arch Linux AUR](https://img.shields.io/badge/AUR-lenger-1793d1?logo=arch-linux)](https://aur.archlinux.org/packages/lenger)

```
┌─ HARBOR DASHBOARD ────────────────────────────────────────────────────────┐
│                                                                           │
│  IMAGES            CONTAINERS          PROJECTS            VOLUMES        │
│  6 total           3 running           1 storefront        4 data vols    │
│  1.7 GB disk       1 stopped           2 services          1.2 GB reclaim │
│                                                                           │
│  CPU & MEMORY TELEMETRY                    RECENT HARBOR EVENTS           │
│  web-1      [ 18.4%]   ▂▄▆█▇▆▄▂ 96.0 MB    12:00:00 container start web-1 │
│  postgres-1 [  7.1%]    ▃▅▆▇▆▅▃ 512 MB     11:58:22 image pull nginx:1.27 │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘
```

---

## Highlights

- **Harbor Instrument Aesthetic:** Calibrated density, marine palette accents, proportional layer bars, and responsive layouts that gracefully scale from **80×24** up to **200×60** ultrawide displays.
- **Image-First Management:**
  - Layer sounding breakdown identifying the `[BIGGEST]` layer command and percentage.
  - Reverse dependency analyzer answering *"Why is this image here?"* across containers and Compose projects.
  - Divergence update detection (`U`) comparing local digests with remote registry manifests.
  - Pluggable vulnerability scanning (`V`) supporting **Trivy**, **Grype**, and **Docker Scout**.
  - Registry search (`S`) and remote tag browser for Docker Hub, GHCR, and Quay.
  - Image tagging (`t`), streaming push (`P`), and tarball export/import (`E` / `I`).
- **Containers & Live Telemetry:**
  - Sparkline CPU & memory utilization curves, port forward mappings, and health checks.
  - Container log viewer with streaming follow (`f`), pause (`p`), timestamps (`t`), regex filter (`/`), and export (`s`).
  - Interactive shell execution (`e`) running `sh`, `bash`, or `ash` inside containers.
  - Batch container actions (start, stop, restart, kill, remove) with explicit confirmations.
- **Docker Compose Orchestration:** Discovers multi-container compose projects and provides live streaming `up`, `down -v`, and `restart` actions.
- **Network & Volume Operations:** List, inspect, and remove unused volumes; create and remove custom Docker bridge networks.
- **Resilient Registry Mirrors:** Transparent pull fallbacks (`mirror.gcr.io`, etc.) and unified diff generator (`M`) for `/etc/docker/daemon.json`.
- **Two-Phase Smart Cleanup:** Visual disk usage sounding (`docker system df`) with selective reclamation previews and mandatory typed safety gates for volumes.
- **Omarchy Desktop Integration:** Automatic terminal detection (`ghostty`, `foot`, `kitty`, `alacritty`), live symlinked theme tracking, and non-conflicting keybinding (`Super+Alt+D`).

---

## Installation

### Arch Linux (AUR)

Build from source:
```sh
yay -S lenger
```

Or install the prebuilt static binary:
```sh
yay -S lenger-bin
```

### Prebuilt Binaries (GitHub Releases)

Download pre-compiled static, CGO-free binaries for Linux (`amd64`, `arm64`, `armv7`) or macOS (`arm64`, `amd64`) from [Releases](https://github.com/Mersad-Moghaddam/lenger/releases).

```sh
curl -fsSL https://github.com/Mersad-Moghaddam/lenger/releases/latest/download/lenger_linux_amd64.tar.gz | tar -xz
sudo install -m 755 lenger /usr/local/bin/
```

### From Source

```sh
git clone https://github.com/Mersad-Moghaddam/lenger.git
cd lenger
go build -trimpath -ldflags="-s -w" -o lenger ./cmd/lenger
sudo install -m 755 lenger /usr/local/bin/
```

---

## Omarchy Desktop Setup

Omarchy users can launch Lenger via Hyprland with a single command:

```sh
# Preview recommended configuration (Super+Alt+D, avoiding lazydocker's Super+Shift+D)
lenger setup-omarchy

# Append non-conflicting binding to ~/.config/hypr/bindings.conf automatically
lenger setup-omarchy --apply
```

Lenger automatically synchronizes live color palettes whenever the active Omarchy theme changes (`~/.config/omarchy/current/theme`).

---

## Usage

```sh
# Launch with Docker daemon
lenger

# Run with bundled theme
lenger --theme omarchy
lenger --theme midnight
lenger --theme harbor

# Run offline fixture demo (never connects to Docker daemon)
LENGER_ENGINE=fake lenger

# Render offline component and screen gallery to stdout
lenger --gallery
lenger --gallery --theme midnight --size 120x40
```

---

## Keyboard Reference

### Navigation & Global Workspaces
| Key | Action |
| --- | --- |
| `1` | Harbor Dashboard & System Overview |
| `2` | Images View & Layer Sounding |
| `3` | Containers & Live Telemetry |
| `4` | Docker Compose Projects |
| `5` | Volumes & Mounts |
| `6` | Networks |
| `7` | System Disk Usage (`df`) & Daemon Info |
| `:` / `Ctrl+P` | Fuzzy Command & Resource Palette |
| `?` | Interactive Help Overlay |
| `q` | Quit or Close Overlay |

### Image Operations (Tab 2)
| Key | Action |
| --- | --- |
| `j` / `k` / `↑` / `↓` | Move cursor through image list |
| `Space` | Select / deselect individual image |
| `Shift+↑` / `Shift+↓` | Multi-select range of images |
| `i` / `Enter` | Toggle inspection, layer breakdown & CVE soundings |
| `p` | Pull new image from registry |
| `S` | Search registries (Docker Hub / GHCR / Quay) & browse tags |
| `U` | Check remote registry update status (manifest digest diff) |
| `V` | Run vulnerability scan (Trivy / Grype / Docker Scout) |
| `t` | Tag selected image (`docker tag`) |
| `P` | Push image to remote registry (`docker push`) |
| `E` | Export image tarball archive (`docker save`) |
| `I` | Import image tarball archive (`docker load`) |
| `r` | Run interactive container from image |
| `d` | Safe image deletion preview |
| `/` | Filter images by repository or tag name |
| `o` | Cycle sort order (name, size, age) |

### Container Operations (Tab 3)
| Key | Action |
| --- | --- |
| `x` | Start selected / highlighted containers |
| `s` | Stop running containers |
| `r` | Restart containers |
| `K` | Force kill running containers |
| `d` | Remove containers (with confirmation) |
| `l` | Stream container logs |
| `e` | Exec interactive shell (`sh`, `bash`, `ash`) |
| `i` | Container inspect (ports, mounts, env, health) |

### Container Logs Viewer
| Key | Action |
| --- | --- |
| `f` | Toggle follow mode (auto-scroll) |
| `p` | Pause / resume log stream |
| `t` | Toggle timestamps display |
| `w` | Toggle text wrapping |
| `/` | Filter log entries with regex |
| `s` | Export log buffer to file |
| `c` | Copy visible log lines to clipboard |

### Compose & Infrastructure (Tabs 4, 5, 6, 7)
| Key | Action |
| --- | --- |
| `u` | `docker compose up -d` on selected project |
| `d` | `docker compose down` (with optional `-v` volume toggle) |
| `r` | `docker compose restart` on selected project |
| `c` | Create Docker network (Tab 6) / Smart Cleanup (Tab 7) |
| `M` | View registry mirrors diff & configuration guide |
| `y` | Copy equivalent `docker` command line to clipboard |

---

## Configuration

Lenger is configured via TOML at `~/.config/lenger/config.toml`:

```toml
[ui]
theme = "omarchy"      # "omarchy", "midnight", "harbor", "monochrome"
ascii = false          # true enforces pure 7-bit ASCII glyphs
reduce_motion = false  # disables spring animations

[registry]
mirrors = [
  "https://mirror.gcr.io",
  "https://registry-1.docker.io"
]

[keys]
pull = "p"
inspect = "i"
delete = "d"
palette = ":"
```

---

## Architecture

Lenger follows a clean hexagonal architecture:

```
cmd/lenger/            CLI entrypoint & flag parsing
internal/
├── engine/            Engine interfaces, classified errors, progress contracts
│   ├── docker/        Official Moby client adapter (ImageList, Logs, Exec, Search)
│   └── fake/          Deterministic fixture engine for offline demos & tests
├── core/
│   ├── store/         Concurrent resource store, filtering, and caching
│   ├── services/      Application facade: vulnerability scanning, mirror rewrites, cleanup
│   └── jobs/          Background job coordinator for image pulls & pushes
├── ui/
│   ├── tokens/        Design tokens, palette definitions, responsive bounds
│   ├── style/         Lip Gloss styler, clean string sanitization, panel wrappers
│   ├── chart/         Unicode sparklines and proportional layer segment bars
│   ├── components/    Atomic badges, pills, dialogs, statusbars
│   ├── views/         Full-screen tabs: images, containers, projects, telemetry
│   └── anim/          Harmonica damped harmonic oscillator spring motion
└── platform/
    └── omarchy/       Hyprland bindings generator & live theme filesystem watcher
```

---

## License

MIT © [Mersad Moghaddam](https://github.com/Mersad-Moghaddam)
