# Lenger: Product Requirements & Design Document

*A beautiful, keyboard-first Docker and image manager for the Omarchy terminal · v0.1 draft · October 21, 2026 · Language: Go · Target: Omarchy (Arch Linux, Hyprland)*

## 1. Name

**Lenger** (from لنگر, “anchor”). It anchors your containers, it is short enough to type a thousand times a day, and it sits comfortably next to Omarchy’s other tool names. Binary: `lenger`.

- Tagline: *Anchor your containers.*
- Why not Berth or Davit: both names are already used by container tools (Berth is an existing Go/Bubble Tea Docker and Podman TUI; Davit is a macOS container UI). Lenger returned no container-tool hits in a quick search, but this is not proof. Before committing, check GitHub, the AUR and pkg.go.dev.
- Runners-up: **Bosun** (the officer who runs the deck), **Gantry** (the crane that moves containers), **Stevedore** (the dockworker).
- Do not put “docker” in the name or logo. Docker’s trademark rules make that a risk.

## 2. Vision and summary

Lenger is a full-screen terminal app for people who live in Omarchy and want to manage Docker without remembering commands. It is **image-first**: pulling, searching, inspecting, updating and cleaning images is as smooth as starting a container. It is **Omarchy-native**: it inherits the active Omarchy theme, opens as a floating window from a keybinding, and feels like part of the OS. It is **beautiful by default**: live sparklines, animated pull progress, rounded panels, status pills, a command palette, spring-physics motion and a braille-resolution chart engine.

One sentence: *the Docker dashboard you would build if Docker Desktop had been designed for a tiling-WM terminal.*

## 3. Problem and market context

Docker is used daily but its CLI is verbose, and the data you need (what is running, what is eating disk, which image is stale, why a pull failed) is spread across five commands. The existing terminal tools are good but leave gaps.

| Tool | What it is | Gap Lenger targets |
| --- | --- | --- |
| lazydocker | Go/gocui TUI, the community default; ships with Omarchy and opens on Super+Shift+D; v0.25.2 released April 2026 | Image workflow is basic (no registry search, tag browser, update check or layer analysis); visual style is functional, not themeable to Omarchy automatically |
| oxker | Rust TUI focused on containers, logs and stats | Containers only; weak image and compose management |
| dry | Veteran Docker TUI | Dated UX, less active |
| Cruise | Go/Bubble Tea TUI with themes, monitoring and vulnerability scanning | Closest competitor in spirit; not Omarchy-aware, no mirror-aware pulling |
| Berth, DockBub, docker-dash, Dockie | Newer Bubble Tea/Textual TUIs | Early stage, narrow feature sets |
| Portainer, Arcane, Dockge | Web UIs | Need a browser and a service; heavy for a local dev box |

**Opportunity.** Nobody owns the intersection of (a) first-class image management, (b) Omarchy-native look and integration, and (c) reliable pulling in hostile network conditions (blocked registries, flaky links, corporate proxies). That intersection is Lenger’s position.

## 4. Users and jobs to be done

| Persona | Context | Top jobs |
| --- | --- | --- |
| **Backend developer** | Runs databases, queues and app containers locally, usually via Compose | Bring a project up/down fast; tail logs; exec into a container; clear disk when it fills up |
| **Homelab / self-hoster** | Many long-running containers, sometimes on a remote box | See health at a glance; update images safely; restart what crashed |
| **DevOps / platform engineer** | Debugs images and registries | Inspect layers and config; compare tags; scan for vulnerabilities; move fast across contexts |
| **Learner** | New to Docker | Discover what exists, what each thing does, and undo mistakes safely |

Core jobs, in priority order: (1) get an image onto this machine, reliably; (2) start, stop and recover containers; (3) understand and reclaim disk; (4) see what is wrong right now; (5) keep images up to date.

## 5. Goals, non-goals and success targets

**Goals**

- Every common Docker task is reachable in two keystrokes or fewer from anywhere.
- The app is visibly the most polished Docker TUI on Linux.
- Safe by default: destructive actions are previewed and confirmed.
- Works offline-first against the local daemon; no accounts, no telemetry.

**Non-goals (v1)**

- Kubernetes, Swarm orchestration or cluster management.
- Replacing `docker build` workflows or being an IDE. (Build support is a v1.x candidate.)
- Hosting a registry or a web UI.
- Windows or macOS support. Linux first, Omarchy first.

**Success targets (to be measured, not guaranteed)**

| Metric | Target |
| --- | --- |
| Cold start to first paint | under 300 ms |
| Idle CPU with 50 containers | under 1% of one core |
| Resident memory | under 60 MB |
| Keystrokes: pull and run a new image | 6 or fewer |
| Crash-free sessions | 99.5% |
| Community signal | listed in Omarchy community resources, AUR package published, 1,000 GitHub stars within 6 months of v1.0 |

## 6. Functional requirements

Priorities: **P0** = MVP, **P1** = v1.0, **P2** = after v1.0.

### 6.1 Images (the heart of the product)

| ID | Requirement | Pri |
| --- | --- | --- |
| IMG-01 | List images with repo, tag, short ID, size, age, and in-use/unused/dangling status; sort and filter | P0 |
| IMG-02 | **Pull** an image by name:tag with per-layer live progress, speed, ETA, cancel, and retry | P0 |
| IMG-03 | Remove one or many images (multi-select), with a warning when containers still use them | P0 |
| IMG-04 | Inspect: config, env (secrets masked), entrypoint, exposed ports, labels, platform, digests | P0 |
| IMG-05 | **Run from image** wizard: name, ports, env, volumes, restart policy, network, with a live preview of the equivalent `docker run` command | P0 |
| IMG-06 | **Search registries** (Docker Hub, GHCR, Quay) with stars, official/verified badges, and a tag browser | P1 |
| IMG-07 | Layer view: size per layer, creating command, and the biggest-layer highlight | P1 |
| IMG-08 | Tag, retag, push, and save/load as tar | P1 |
| IMG-09 | **Update check**: compare local digest with the registry digest for the same tag; show “update available”; one-key pull-and-recreate | P1 |
| IMG-10 | Vulnerability summary via an installed scanner (Trivy, Grype or Docker Scout); severity counts per image | P1 |
| IMG-11 | “Why is this image here?” view: containers, Compose projects and parent images that depend on it | P1 |
| IMG-12 | History/diff between two tags of the same repo (size and layer changes) | P2 |
| IMG-13 | Build from a Dockerfile with streaming output and BuildKit progress | P2 |

### 6.2 Containers

| ID | Requirement | Pri |
| --- | --- | --- |
| CTR-01 | List with state pills, image, ports, uptime, CPU and memory sparklines | P0 |
| CTR-02 | Start, stop, restart, pause/unpause, kill, remove; bulk actions on a selection | P0 |
| CTR-03 | Live logs: follow, pause, search, filter by level or regex, timestamps toggle, wrap toggle, copy and save | P0 |
| CTR-04 | Exec into a container (suspend the TUI, run `docker exec -it`, restore on exit); choose shell | P0 |
| CTR-05 | Inspect with a structured view (not raw JSON first), raw JSON available | P0 |
| CTR-06 | Live stats: CPU, memory, network and block I/O, with history charts | P1 |
| CTR-07 | Health check status and last probe output | P1 |
| CTR-08 | Edit and recreate (change ports, env, limits) | P2 |
| CTR-09 | Copy files to and from a container | P2 |

### 6.3 Compose projects

| ID | Requirement | Pri |
| --- | --- | --- |
| PRJ-01 | Auto-discover projects from container labels; group containers by project | P0 |
| PRJ-02 | **Up / Down / Restart / Pull / Recreate** per project and per service, with streamed output | P0 |
| PRJ-03 | Merged, color-coded log stream across a project’s services | P1 |
| PRJ-04 | Register project directories explicitly so stopped projects appear too | P1 |
| PRJ-05 | Profiles and env-file selection; `down -v` behind a strong confirmation | P1 |

### 6.4 Volumes, networks, system

| ID | Requirement | Pri |
| --- | --- | --- |
| VOL-01 | List volumes with size, driver, and which containers use them; remove unused | P0 |
| VOL-02 | Browse volume contents read-only through a helper container; export a volume | P2 |
| NET-01 | List networks, attached containers, subnets; create, remove, connect, disconnect | P1 |
| SYS-01 | Disk usage breakdown (images, containers, volumes, build cache) with a bar chart | P0 |
| SYS-02 | **Smart cleanup**: preview exactly what will be removed and how much space returns, then confirm | P0 |
| SYS-03 | Daemon info, version, API version, storage driver, contexts, resource totals | P1 |
| SYS-04 | Live event feed from the Docker events API | P1 |

### 6.5 Registries, mirrors and network resilience

This is a real pain point for users in restricted regions or behind corporate proxies, and few tools address it.

| ID | Requirement | Pri |
| --- | --- | --- |
| REG-01 | Classify pull failures (auth, rate limit, not found, TLS, DNS, timeout, blocked or forbidden) and show a plain-language cause and next step | P0 |
| REG-02 | **Mirror-aware pull**: configurable mirror list; on failure offer “retry via mirror” that rewrites the reference (for example, `mirror.example/library/nginx:1.27`) and, if desired, retags to the original name | P1 |
| REG-03 | Guided editor for the daemon’s `registry-mirrors` and proxy settings with a diff preview. Writing `/etc/docker/daemon.json` needs root and a daemon restart; Lenger shows the change and asks before using sudo, never silently | P1 |
| REG-04 | Respect `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` and Docker credential helpers; never store registry credentials itself | P0 |
| REG-05 | Resumable, concurrent-safe pulls with a visible job queue | P1 |
| REG-06 | Log in to / out of registries through the standard Docker credential store | P1 |

### 6.6 Platform features

| ID | Requirement | Pri |
| --- | --- | --- |
| PLT-01 | **Command palette** (`:` or Ctrl+P): fuzzy search over every action and every resource | P0 |
| PLT-02 | Global filter (`/`) on any list; saved filters | P0 |
| PLT-03 | Multi-select with Space, range select with Shift | P0 |
| PLT-04 | Docker contexts and `DOCKER_HOST` (including `ssh://`) with a context switcher | P1 |
| PLT-05 | Podman via its Docker-compatible socket | P2 |
| PLT-06 | Custom commands and keybindings in config | P1 |
| PLT-07 | Copy-as-command: every action can show and copy the equivalent CLI command (teaches Docker to learners) | P1 |
| PLT-08 | Job tray: all long operations (pulls, prunes, up/down) visible, cancellable, with history for the session | P0 |

## 7. UX and visual design

### 7.1 Principles

1. **Show state, not menus.** The first screen answers “is everything OK?” without any keypress.
2. **Two keystrokes to anywhere.** Number keys switch tabs; the palette reaches everything else.
3. **Never surprise.** Destructive actions preview their effect; the equivalent CLI command is always one key away.
4. **Omarchy is the design system.** Colors come from the active theme; the app never ships a competing palette.
5. **Degrade gracefully.** Works in 80×24 with plain ASCII and no color; shines in a large Ghostty/Alacritty window with Nerd Fonts.

### 7.2 Design ambition

Lenger must look like a finished product, not a utility. Every screen clears three bars:

1. **Screenshot-worthy in two seconds.** The dashboard, the images tab and the pull overlay each work as a README hero image with no editing.
2. **Alive, not busy.** Motion explains change (a row glides, a number counts up, a bar fills). Nothing moves for decoration while the app is idle.
3. **One system.** Every color, spacing unit, glyph and animation comes from design tokens. No view invents its own look.

Benchmark: the visual polish of the best Charm-built apps, the information density of btop, and lazydocker’s breadth of features, in one interface.

### 7.3 Design toolchain

Use the Charm stack as the base, add purpose-built libraries where it stops, and make the design testable.

| Concern | Tool | How Lenger uses it |
| --- | --- | --- |
| Event loop and rendering | Bubble Tea v2 (`charm.land/bubbletea/v2`) | Elm-style model per view; one `View` per frame; mouse and keyboard input |
| Styling and layout | Lip Gloss v2 (`charm.land/lipgloss/v2`, with `/table`, `/tree`, `/list`) | Cards, borders, joins, gradients and color blending; **layers and compositor** for modals, toasts and dimmed backdrops; `tree` for the inspect view |
| Widgets | Bubbles v2 (`charm.land/bubbles/v2`) | Viewport (logs), text input, spinner, progress, help, paginator as the base for custom components |
| Forms and wizards | Huh v2 (`charm.land/huh/v2`) | Run-container wizard, registry login, mirror editor |
| Markdown | Glamour v2 (`charm.land/glamour/v2`) | In-app help, release notes, image descriptions from registries |
| Logging | Log v2 (`charm.land/log/v2`) | Debug log to the state directory (never to the screen) |
| Animation | Harmonica (`charmbracelet/harmonica`) | Spring physics for selection glide, panel slides, count-up numbers, toast entry |
| Charts | Internal `chart` package (braille and eighth-block rendering); optionally ntcharts | Sparklines, time-series with area fill, bars, treemap. ntcharts provides sparklines, bar charts and braille time-series, but confirm Bubble Tea v2 compatibility before depending on it; the internal package is about 300 lines and gives full control |
| Mouse hit-testing | Internal zone registry (or BubbleZone if v2-compatible) | Clickable tabs, rows, buttons, scrollbar dragging |
| Color profile | `colorprofile` and Lip Gloss v2 downsampling | Truecolor by default, automatic fallback to 256 and 16 colors |
| Fuzzy search | A fuzzy matcher such as `sahilm/fuzzy` | Command palette and list filter with match highlighting |
| Recording and visual QA | VHS and Freeze | Scripted tapes produce README GIFs and golden frames; Freeze makes PNG/SVG screenshots of any view |
| Graphics (optional, P2) | Kitty graphics protocol or Sixel, behind capability detection | Logo and treemap upgrade on terminals that support images; text fallback everywhere else |

Version notes. A third-party registry listing shows Bubble Tea v2.1.0, Lip Gloss v2.0.6, Bubbles v2.2.1, Huh v2.0.3, Glamour v2.0.1 and Log v2.0.1; run `go get` with `@latest` and pin whatever resolves. Use only the `charm.land/*/v2` import paths, because the older `github.com/charmbracelet/*` paths are the pre-v2 APIs. Bubble Tea, Bubbles, Huh and Lip Gloss share types and must be upgraded together.

### 7.4 Design language

**Semantic tokens.** Views reference roles, never hex values. Roles are filled from the active theme.

| Token | Role |
| --- | --- |
| `bg` / `surface` / `raised` | Three elevation levels: app background, cards, overlays |
| `border` / `border-focus` | Idle and focused panel outlines |
| `text` / `muted` / `faint` | Primary, secondary and disabled text |
| `accent` / `accent2` | Brand and focus color, plus the second stop of the signature gradient |
| `success` / `warning` / `danger` / `info` | Status colors, always paired with a glyph |
| `series1` to `series6` | Chart and layer colors derived from the theme’s accents so charts match the theme |

**Rules**

- **Gradient discipline.** The accent gradient (`accent` to `accent2`, blended in a perceptual color space) appears on exactly three things: the wordmark, the active-tab underline, and progress fills. Everything else is flat, which makes the gradient feel special.
- **Elevation by tone, not only lines.** Cards sit on `surface`, overlays on `raised` over a dimmed backdrop (Lip Gloss layers). Borders are one rounded weight; focus changes color, not thickness.
- **Pills and badges.** Rounded caps (Nerd Font half-circle glyphs) with an ASCII fallback in brackets. A status is always glyph plus color: `● running`, `○ exited`, `◐ paused`, `✕ unhealthy`, `⇡ update`.
- **Typography by weight and case.** One terminal font. Hierarchy uses bold, dim and spaced uppercase muted section labels (`I M A G E S`). Italics only for hints.
- **Sub-cell precision.** Bars use eighth-block characters (`▏▎▍▌▋▊▉█`) so a fill moves in 8 steps per cell. Charts use braille (2×4 dots per cell) for smooth curves.
- **Icons.** One Nerd Font icon per resource type (image, container, volume, network, project, health) from a single table, with an ASCII table behind the `ascii_icons` option.
- **Density.** Comfortable by default (padded cards); a compact, btop-style mode on `D`.
- **Spacing.** One unit grid (1 cell vertical, 2 cells horizontal) defined in tokens. No ad-hoc padding.

### 7.5 Signature screens

The three wireframes below are the contract for the first design pass. Exact glyphs may vary with the font.

**Dashboard (bento grid)**

```
 ⚓ LENGER   Dashboard  Images  Containers  Projects  Volumes  Networks  System
 ━━━━━━━━━━━                                              ● docker 29 · local
╭─ HEALTH ──────────╮ ╭─ CPU  18% ────────────────╮ ╭─ MEMORY  2.4 / 16 GB ─────╮
│  12  ● running    │ │ ⢀⣠⣴⣿⣿⣷⣦⣄⣀⣠⣴⣾⣿⣿⣶⣤⣀⢀⣀⣠⣤⣶ │ │ ⣀⣀⣠⣤⣤⣴⣴⣶⣶⣶⣶⣾⣾⣿⣿⣿⣿⣷⣶ │
│   3  ○ exited     │ │ postgres 6% · web 4%      │ │ postgres 1.1 GB · redis…  │
│   1  ✕ unhealthy  │ ╰───────────────────────────╯ ╰───────────────────────────╯
╰───────────────────╯ ╭─ DISK  5.8 GB · 2.1 GB reclaimable ─────────────────────╮
╭─ NEEDS ATTENTION ─╮ │ ▐██████████████████▌▐███████▌▐███▌▐░░░░░░░░░░░░░░░░░░ │
│ ✕ api-1 unhealthy │ │  images 3.2 GB     volumes 1.9 GB     cache 0.7 GB     │
│ ⇡ 4 image updates │ ╰────────────────────────────────────────────────────────╯
│ ♻ 2.1 GB to clean │ ╭─ EVENTS ───────────────────────────────────────────────╮
╰───────────────────╯ │ 12:41:07 ● web-1 started      12:40:52 ⇣ pulled redis:7.4 │
                      ╰────────────────────────────────────────────────────────╯
 ? help   : palette   c cleanup   p pull                              ⇣ 0 jobs
```

**Images (master/detail with layer ribbon)**

```
╭─ IMAGES · 24 · 3.2 GB ─────────────────╮╭─ nginx:1.27-alpine ──────────────────────╮
│ ▌ nginx        1.27-alpine   48 MB   ⇡ ││ sha256:a1b2c3…  linux/amd64  3 weeks old │
│   postgres     16           432 MB   ● ││ ⇡ Update available   1.27.3 → 1.27.5     │
│   redis        7.4          117 MB   ● ││ ◆ Scan  0 critical · 2 high · 11 medium  │
│   golang       1.23         843 MB     ││                                          │
│   node         22-slim      241 MB     ││ L A Y E R S                              │
│   ubuntu       24.04         78 MB   ○ ││ ▐████████████▌▐███▌▐▌▐██▌   48 MB        │
│   …                                    ││  ▸ 42 MB  RUN apk add --no-cache nginx…  │
│                                        ││    6 MB   COPY nginx.conf /etc/nginx/…   │
│                                        ││ U S E D   B Y   web-1 ● running  proxy-2 ○│
╰────────────────────────────────────────╯╰──────────────────────────────────────────╯
 p pull   r run   u update   d delete   s scan   / filter   space select    ⇣ redis 62%
```

**Pull overlay (modal over a dimmed backdrop)**

```
            ╭─ Pull image ─────────────────────────────────────╮
            │  ⇣ redis:7.4                  62%   12.4 MB/s · 9 s │
            │  ████████████████▌░░░░░░░░░░  (gradient fill)       │
            │                                                     │
            │  a3f1c9e2b7d4  ██████████████████████  done         │
            │  7c0d8e5a1f92  ████████████▌░░░░░░░░░  58%  ⣀⣤⣶⣿⣷  │
            │  e19b44c07a63  ░░░░░░░░░░░░░░░░░░░░░  waiting       │
            │                                                     │
            │  esc hide (keeps pulling)    x cancel               │
            ╰─────────────────────────────────────────────────────╯
```

Two more overlays follow the same pattern. The **command palette** is a centered `raised` overlay with fuzzy results (icon, name, match highlights), a recent-actions group, and a preview line showing the equivalent CLI command. The **cleanup preview** is a checklist of categories with reclaimable sizes and a live total at the bottom.

### 7.6 Signature components

- **Wordmark header.** Gradient anchor wordmark, tab bar with an underline that slides between tabs, daemon status chip with a pulsing dot when connected.
- **Stat cards.** Large numerals, a delta arrow versus the last minute, and a braille sparkline.
- **Braille time-series.** CPU and memory with area fill, a 60-second window by default, hover or cursor readout.
- **Layer ribbon.** One stacked bar per image where each layer is a segment colored from `series1` to `series6`; the cursor on a layer shows its command and size, and the largest layer is outlined.
- **Disk treemap (P1).** A squarified treemap of images, volumes and cache by size, drawn with half-block cells; selecting a tile jumps to the resource.
- **Log viewer.** Level-colored lines, JSON auto-detected and pretty-printed with folding, timestamp gutter, search highlights, and a scrollbar minimap with markers at errors.
- **Inspect tree.** Collapsible tree (Lip Gloss `tree`) with secret values masked and a reveal key.
- **Job tray and toasts.** A compact tray in the footer with per-job mini progress; toasts slide in, stack, and auto-dismiss.
- **Heartbeat dot.** Running containers show a slow two-frame pulse; static when `reduce_motion` is on.
- **Empty states.** A small anchor-and-waves drawing with the single next action.
- **Splash (optional).** A sub-400 ms gradient wordmark reveal on start; skippable with any key and disabled in config.

### 7.7 Motion system

| Motion | Behavior | Duration |
| --- | --- | --- |
| Selection glide | The highlight bar springs to the new row (Harmonica) instead of jumping | about 120 ms |
| Tab underline | Slides and resizes to the new tab | about 180 ms |
| Count-up | Numbers ease from the old value to the new one | about 300 ms |
| Progress fill | Eased fill with a soft moving highlight on the gradient | continuous while active |
| Overlay in/out | Backdrop dims by blending toward `bg`; the overlay fades in by color blend | about 120 ms |
| Toast | Slides in from the bottom edge, holds 3 s, slides out | about 200 ms |
| Heartbeat | Pulse on running-state dot | 1.6 s loop |

Rules:

- **Idle means zero ticks.** Frame ticks run only while an animation is active, at 30 fps by default and up to 60 fps on request. A static screen schedules no timers.
- **Budget.** CPU stays under 5% of one core during animation and under 1% idle.
- **Reduce motion.** `reduce_motion = true`, `NO_MOTION=1`, or a slow-terminal heuristic (for example over SSH) replaces all animation with instant state changes.
- **No flicker.** One composed frame per tick; overlays use the compositor so the backdrop is never redrawn piecemeal.

### 7.8 Themes, responsiveness and accessibility

- **Theme sources, in order:** the active Omarchy theme (live), an explicit `theme` in config, then a bundled theme. Ship hand-tuned bundled themes for popular palettes (for example Tokyo Night, Catppuccin Mocha and Latte, Gruvbox, Nord, Rose Pine, Kanagawa) plus a house theme, **Lenger Midnight**, designed first and used for all marketing screenshots.
- **Contrast.** Automated test: text tokens must reach at least 4.5:1 against the surface they are drawn on in every bundled theme; failing themes are rejected in CI.
- **Color depth.** Truecolor by default; automatic downsampling to 256 and 16 colors; `NO_COLOR` honored; a high-contrast theme included.
- **Meaning never depends on color alone.** Every status has a glyph and a word.

| Width | Layout |
| --- | --- |
| under 80 | Minimum-size message |
| 80 to 119 | Single column; detail slides in over the list |
| 120 to 179 | Master/detail |
| 180 and up | Three panes with live charts beside the list |

### 7.9 Design quality gates

These are checked by automation so an autonomous agent can verify design work without a human review step.

1. **Token-only styling.** A lint step fails the build if a view contains a hard-coded color or ad-hoc padding outside the tokens package.
2. **Gallery.** `lenger --gallery` renders every component and screen with fixture data, in every bundled theme, at four terminal sizes, without needing Docker.
3. **Golden frames.** The gallery output is snapshot-tested; unintended visual diffs fail CI.
4. **Layout assertions.** Tests check that no line exceeds the terminal width, panels align, and overlays stay inside the screen.
5. **Contrast and color-depth tests** as described in 7.8.
6. **Motion tests.** Idle state schedules no ticks; `reduce_motion` produces no animation messages.
7. **VHS tapes** for the dashboard, images tab, pull flow, cleanup flow and palette generate the README GIFs and double as smoke tests.

### 7.10 Key flows

**Pull and run a new image**

1. On the Images tab press `p`. A prompt opens with an autocomplete drawn from recent pulls and registry search.
2. Type `redis`, press Tab to browse tags, pick `7.4`, press Enter.
3. The job tray shows layer-by-layer progress. On failure, the error is classified and a mirror retry is offered.
4. When complete, a toast offers “Run now (r)”. The run wizard opens with sensible defaults and the live `docker run` preview.

**Clean up disk safely**

1. Press `c` anywhere to open Cleanup. Lenger lists categories: dangling images, unused images, stopped containers, unused volumes, build cache, each with reclaimable size.
2. Toggle categories. A preview table shows each item and the total space returned.
3. Confirm. Volumes require typing the word `volumes` because the data cannot be recovered.

**Bring a Compose project up or down**

1. Projects tab, select a project, press `u` (up) or `d` (down). Output streams in a split pane.
2. `down` never removes volumes unless `-v` is explicitly toggled and confirmed.

### 7.11 Default keybindings

| Key | Action | Key | Action |
| --- | --- | --- | --- |
| `1`–`7` | Switch tab | `/` | Filter list |
| `j` `k` / arrows | Move | `:` or Ctrl+P | Command palette |
| `Enter` | Open detail | `Space` | Select / multi-select |
| `p` | Pull image | `r` | Run image / restart container (contextual) |
| `s` | Stop | `x` | Start |
| `d` | Delete / down (contextual) | `u` | Up (project) / update (image) |
| `l` | Logs | `e` | Exec shell |
| `i` | Inspect | `y` | Copy equivalent CLI command |
| `c` | Cleanup | `?` | Help overlay |
| `Tab` | Switch pane | `q` / Esc | Back / quit |

Vim-style and arrow keys both work. All bindings are remappable.

### 7.12 States and edge cases

- **Daemon not running or socket permission denied:** a friendly full-screen state with the exact fix (start the service, add the user to the `docker` group) and a retry key.
- **Empty states** explain the next action (“No images yet. Press `p` to pull one.”).
- **Terminal too small:** show a minimum-size message instead of a broken layout.
- **Long-running or failed jobs** never block the UI; errors appear in the tray with details and a retry key.

## 8. Omarchy integration

These points need verification against the current Omarchy release before implementation, because Omarchy’s internals change between versions.

- **Theme adapter:** locate the active theme under the user’s Omarchy config directory and read its palette (a palette file if present, otherwise parse the terminal theme file). Isolate this behind a `ThemeSource` interface so a layout change only affects one adapter.
- **Launching:** use Omarchy’s own helper for opening a TUI in a floating, centered terminal window, so Lenger gets the same window rules as other Omarchy TUIs.
- **Keybinding:** Omarchy already binds Super+Shift+D to lazydocker, so Lenger should suggest a different default (for example, Super+Alt+D) and print the exact `bindings.conf` line. It must never overwrite user config; `lenger setup-omarchy` only prints or appends with confirmation.
- **Menu and launcher entry:** install a `.desktop` file so Lenger appears in the app launcher.
- **Fonts:** assume the Nerd Font Omarchy ships, but detect and fall back to ASCII icons.

## 9. System design

### 9.1 Non-functional requirements

| Area | Requirement |
| --- | --- |
| Performance | Cold start under 300 ms; UI input latency under 50 ms; list views stay smooth with 1,000+ images or containers (virtualized rendering) |
| Resource use | Idle CPU under 1%; memory under 60 MB; stats streams only for visible or selected rows |
| Reliability | Reconnect automatically if the daemon restarts; no state that can be corrupted; all jobs cancellable |
| Security | The Docker socket is root-equivalent: no network listeners, no telemetry, no secrets in logs, env values masked by default, credentials only through Docker’s credential helpers |
| Compatibility | Linux x86\_64 and arm64; Docker Engine with API version negotiation (a recent lazydocker release documents Docker 29+ as its requirement, so test against current and previous major versions); Compose v2 plugin |
| Distribution | Single static binary (CGO disabled), AUR packages, GitHub Releases |
| Maintainability | Engine behind an interface; UI never calls the Docker SDK directly |

Constraints: small team (likely one developer plus coding agents), Go as the implementation language, and no daemon or service of its own.

### 9.2 High-level architecture

```
┌──────────────────────── Lenger process (single binary) ────────────────────────┐
│ UI layer (Bubble Tea)                                                           │
│  ├─ App model   router · layout · command palette · toasts · job tray           │
│  ├─ Views       dashboard · images · containers · projects · volumes · system   │
│  └─ Theme       Omarchy adapter → semantic palette → Lip Gloss styles           │
│        ▲ tea.Msg                                  │ tea.Cmd                     │
│ Application core                                   ▼                             │
│  ├─ Store     normalized in-memory cache + indices (image→containers, …)        │
│  ├─ Jobs      queue · cancellation · progress streams (pull, prune, up/down)    │
│  └─ Services  ImageSvc · ContainerSvc · ProjectSvc · RegistrySvc · CleanupSvc   │
│        ▲                                                                        │
│ Engine abstraction (interface)                                                  │
│  ├─ DockerEngine   Moby client over unix socket / ssh / tcp                     │
│  ├─ ComposeRunner  shells out to `docker compose`, parses JSON output           │
│  └─ PodmanEngine   Docker-compatible socket (P2)                                │
│ Optional adapters: Trivy / Grype / Scout CLI · Registry HTTP (Hub, GHCR, Quay)  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 9.3 Technology choices

| Concern | Choice | Notes |
| --- | --- | --- |
| Language | Go | Single static binary, strong Docker SDK, matches the author’s primary stack |
| TUI stack | Bubble Tea v2, Lip Gloss v2, Bubbles v2, Huh v2, Glamour v2 (all `charm.land/*/v2`) | Layers and compositor for overlays, color blending for gradients. Versions and the upgrade-together rule are in 7.3; pin what `go get @latest` resolves |
| Animation | Harmonica | Spring physics for glide, count-up and panel motion; ticks only while animating |
| Charts | Internal `chart` package (braille and eighth-block); ntcharts optional | Full control over look; adopt ntcharts only if it works with Bubble Tea v2 |
| Docker access | Official Moby Go client | Verify the current module path (it has been moving); use API version negotiation |
| Compose | Shell out to `docker compose` with JSON output | Exact behavior parity with the user’s CLI; plugin required |
| Config | TOML at `~/.config/lenger/config.toml` | Human-editable, easy to document |
| Persistence | In-memory plus a small JSON state file (recent pulls, saved filters, registered projects) | No database in v1 |
| Release | GoReleaser and GitHub Actions | Builds binaries, checksums, AUR metadata |
| Tests | `go test`, model tests driving `Update` with constructed messages, full-program runs with an ASCII color profile, gallery golden frames, VHS tapes | See 7.9 and 11.2 |

### 9.4 Data flow

1. **Startup:** connect to the engine, fetch initial lists in parallel, paint immediately with loading placeholders, then fill in.
2. **Steady state:** subscribe to the Docker events stream. Each event invalidates only the affected resource in the Store and triggers a targeted refresh. A slow safety poll (every 30 s) catches missed events.
3. **Stats and logs:** opened lazily for the visible or focused container only, closed when it scrolls out of view or the pane closes. Logs use a capped ring buffer (default 50,000 lines) with back-pressure.
4. **Actions:** the UI emits a command, the Jobs manager runs it with a cancellable context, progress arrives as messages, and the Store updates from resulting events.
5. **Pulls:** the image pull JSON stream is parsed into per-layer progress records and aggregated into one job with speed and ETA.

### 9.5 Core data model (sketch)

```go
type ImageView struct {
    ID, ShortID   string
    RepoTags      []string
    Digests       []string
    Size, Shared  int64
    Created       time.Time
    Dangling      bool
    UsedBy        []ContainerRef
    UpdateState   UpdateState // Unknown | UpToDate | Available | CheckFailed
    Vulns         *VulnSummary // nil when not scanned
}

type Job struct {
    ID        string
    Kind      JobKind // Pull, Remove, Prune, ComposeUp, ComposeDown, ...
    Target    string
    State     JobState // Queued | Running | Succeeded | Failed | Cancelled
    Progress  Progress // bytes, layers, percent, speed, eta
    Err       *ClassifiedError // category, human message, suggested action
    StartedAt time.Time
}

type ClassifiedError struct {
    Category  ErrCategory // Auth | RateLimit | NotFound | TLS | DNS | Timeout | Blocked | Daemon | Perm | Unknown
    Message   string
    Suggest   []Action    // e.g. RetryViaMirror, OpenDaemonConfig, Login
}
```

### 9.6 Caching strategy

| Data | Source | Invalidation |
| --- | --- | --- |
| Resource lists | Engine API | Docker events, plus a 30 s safety poll |
| Container stats | Streaming API | Only while visible; dropped on blur |
| Registry tag lists and search | Registry HTTP | 10-minute TTL, manual refresh with `R` |
| Update-check results | Registry digest lookup | 6-hour TTL, per image, with jittered background checks |
| Vulnerability scans | Scanner CLI | Keyed by image digest; cached until the digest changes |

### 9.7 Error handling and retries

- Event stream drops: exponential backoff (1 s to 30 s) with a visible “reconnecting” indicator; full refresh on reconnect.
- Pull failures: classify, show cause and action; offer retry (same registry), retry via mirror, or login.
- Update checks and registry calls: short timeouts (5 to 10 s), never block the UI, failures show as “check failed” and not as errors.
- Panics: recover at the job boundary, write a crash report to the state directory, restore the terminal cleanly.
- Partial failures in bulk actions are listed per item; successes are kept.

### 9.8 Scale and reliability

- **Load estimate:** a heavy developer machine has up to a few thousand images and a few hundred containers. The design target is 5,000 images and 1,000 containers with smooth scrolling, via windowed list rendering and incremental store updates.
- **Remote hosts:** the same engine interface over an SSH context. Latency is hidden with optimistic loading states and a longer safety poll interval.
- **No single point of failure inside Lenger:** it holds no authoritative state. Killing it at any moment cannot corrupt Docker.
- **Observability for the developer:** `--debug` writes structured logs to the state directory; a hidden diagnostics view shows event rate, goroutine count and cache sizes.

### 9.9 Trade-off decisions

| Decision | Chosen | Alternative | Reason and cost |
| --- | --- | --- | --- |
| UI framework | Bubble Tea | gocui (lazydocker), Rust ratatui | Go fits the author’s stack and Bubble Tea has the best styling tools. Cost: more hand-written layout code than a panel framework |
| Compose control | Shell out to CLI | compose-go library | Guarantees behavior parity and handles profiles and env files. Cost: depends on the plugin being installed |
| Persistence | In-memory plus JSON | SQLite | Nothing in v1 needs queries or history. Revisit if usage history or analytics is added |
| Update checks | Digest comparison against the registry | Poll tags only | Digest comparison is the only reliable test for mutable tags. Cost: registry rate limits, hence the TTL and jitter |
| Vulnerability scan | Delegate to installed scanner | Embed a scanner | Keeps the binary small and avoids shipping a vulnerability database. Cost: optional dependency |
| Daemon config edits | Guided with diff and explicit sudo | Silent privileged write | Trust and safety over convenience |
| Theme source | Read Omarchy theme with a fallback | Own theme system only | Core to the Omarchy-native promise. Cost: an adapter that must track Omarchy changes |

## 10. Packaging, distribution and publishing

- **GitHub Releases** with static binaries for x86\_64 and arm64, checksums, and signed tags (GoReleaser).
- **AUR:** publish `lenger` (build from source) and `lenger-bin` (prebuilt). Arch is Omarchy’s base, so this is the primary install path.
- **Omarchy visibility:** check the current Omarchy documentation and community channels for the supported way to list or submit extensions and tools, and follow it. Do not assume a specific marketplace exists.
- **Discovery:** submit to Terminal Trove, relevant “awesome” lists (TUI, Docker, Omarchy), and post a short demo GIF.
- **Docs site:** a README with a hero GIF, a 60-second quick start, and a keybinding cheat sheet. Add a CONTRIBUTING guide and a theme-adapter guide.
- **Versioning:** semantic versioning, a changelog, and a clear compatibility matrix (Docker, Omarchy).

## 11. Delivery plan, testing and agent hand-off

### 11.1 Milestones with acceptance criteria

Each milestone ends with automated checks an autonomous agent can run to verify and self-correct, so no human review checkpoint is needed to know whether the milestone is done.

| Milestone | Scope | Done when |
| --- | --- | --- |
| **M0 Foundations** | Repo, CI, Engine interface with fake and real Docker implementations, theme adapter with fallback, App shell, tabs, help overlay, design-token package, component gallery (lenger --gallery), first VHS tapes | `go build`, `go vet`, lint and unit tests pass; app starts against a fake engine and a real daemon; theme switches live |
| **M1 MVP** | IMG-01 to IMG-05, CTR-01 to CTR-05, VOL-01, SYS-01, SYS-02, PLT-01 to PLT-03, PLT-08, REG-01, REG-04 | An integration test pulls an image, runs it, reads logs, execs, stops, removes, and cleans up against real Docker in CI; model tests pass |
| **M2 Projects and polish** | PRJ-01 to PRJ-05, CTR-06, CTR-07, NET-01, SYS-03, SYS-04, dashboard | Compose up/down test on a sample project passes; golden-file view tests pass at 80×24, 120×40 and 200×60 |
| **M3 Registry power** | IMG-06 to IMG-11, REG-02, REG-03, REG-05, REG-06 | Update check detects a changed tag against a local test registry; mirror retry test passes; scanner adapter parses sample reports |
| **M4 Release** | Packaging, Omarchy setup command, docs, vhs recordings, performance pass | Performance budget met on a 5,000-image fixture; AUR build succeeds in a clean Arch container; README GIFs generated by CI |
| **M5 v1.x** | PLT-04 to PLT-07 hardening, Podman, build support, CTR-08, VOL-02, IMG-12, IMG-13 | Defined after v1.0 feedback |

### 11.2 Testing strategy

- **Unit:** services and the Store against a fake engine with scripted event streams.
- **UI:** model tests that drive Update with constructed messages, plus full-program runs with tea.WithInput, tea.WithOutput, tea.WithWindowSize and an ASCII color profile for stable strings; golden-file snapshots of rendered views at several terminal sizes.
- **Integration:** real Docker in CI with small images; a local test registry for update-check and mirror tests.
- **Visual:** vhs tapes produce the README recordings and double as smoke tests.
- **Resilience:** chaos tests that stop the daemon mid-stream, drop the network mid-pull, and send malformed JSON.

### 11.3 Repository docs layout for agent-driven development

Keep a single source of truth in the repo and let agent prompts point to it instead of repeating it:

- `docs/PRD.md` (this document: requirements and acceptance criteria)
- `docs/ARCHITECTURE.md` (sections 9 and 10 expanded as the code evolves)
- `docs/UX.md` (layout, keybindings, visual language)
- `CLAUDE.md` (build, test and lint commands, coding conventions, and the instruction to read the docs above before each task)

A task prompt then reduces to: *“Implement milestone M1 as defined in docs/PRD.md section 11.1. Run the listed checks and fix failures until they pass.”*

## 12. Risks and mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Crowded market (lazydocker is bundled with Omarchy) | Low adoption | Lead with the image workflow, Omarchy-native polish, and mirror-aware pulling; ship a short comparison and demo GIF |
| Omarchy internals change (theme files, launcher helpers) | Broken theme sync | Adapters behind interfaces, fallback theme, compatibility matrix, CI against the latest Omarchy defaults where feasible |
| Docker API or SDK changes | Breakage | API version negotiation, test against current and previous Docker majors, isolate the SDK behind the Engine interface |
| Bubble Tea major-version migration | Rework | Pin versions, keep framework calls inside the UI layer |
| Registry rate limits or blocks | Failed update checks and pulls | TTL caches, jitter, mirror fallback, clear error classification |
| Destructive-action mistakes | Data loss and lost trust | Previews, typed confirmation for volumes, equivalent-command display |
| Terminal rendering differences | Visual glitches | Test in Omarchy’s default terminal at several sizes; ASCII fallback mode |

## 13. Open questions

1. Final name: confirm availability of Lenger on GitHub, AUR and pkg.go.dev, or choose Bosun, Gantry or Stevedore.
2. Which scanner should be the first-class integration (Trivy, Grype or Docker Scout)?
3. Should v1.0 include image building, or stay focused on pull, run and manage?
4. Which license (MIT is typical for this ecosystem)?
5. Is a plugin or custom-command system needed in v1.0, or is config-based custom commands enough?
6. What is the exact supported Omarchy version range for the setup command?

## 14. What to revisit as it grows

- A real persistence layer (SQLite) if pull history, usage analytics or fleet views are added.
- A background agent mode for multi-host monitoring, which would change the “no service” stance.
- A plugin API if the community wants extensions.
- Podman and containerd as first-class engines.
- Kubernetes support, which is intentionally out of scope for now.

## Appendix A: Example configuration

```toml
# ~/.config/lenger/config.toml
[ui]
theme = "omarchy"        # "omarchy" follows the active Omarchy theme
ascii_icons = false
reduce_motion = false
mouse = true

[docker]
context = ""             # empty = current docker context / DOCKER_HOST
safety_poll_seconds = 30

[registry]
mirrors = ["mirror.example.com"]
update_check = true
update_check_ttl_hours = 6
scanner = "auto"         # auto | trivy | grype | scout | none

[logs]
buffer_lines = 50000
show_timestamps = false

[keys]
# remap any default, for example:
# pull = "P"
```

## Appendix B: Sources checked (October 2026)

- Omarchy manual, TUIs chapter (lazydocker on Super+Shift+D): learn.omacom.io
- lazydocker releases and packaging: github.com/jesseduffield/lazydocker, the Arch Linux package page, and a third-party keybinding reference
- Competing tools: oxker (github.com/mrjackwills/oxker), Cruise (github.com/NucleoFusion/cruise), Berth (github.com/rluders/berth), DockBub, docker-dash, Dockie, Davit
- Alternatives overview: alternativeto.net pages for lazydocker and oxker

## Appendix C: Claude Code prompt for the project skeleton (milestone M0)

How to use it:

1. Create an empty Git repository and export this document as Markdown to `docs/PRD.md`.
2. Open Claude Code at the repository root and paste the prompt below.
3. The prompt points to the PRD instead of repeating it, and ends in a self-verification loop, so no review checkpoint is needed between steps.

```text
ROLE
You are the lead engineer bootstrapping "Lenger": a Go terminal app (TUI) that manages Docker and images on Omarchy (Arch Linux, Hyprland). Work autonomously. Do not stop to ask questions: make the best decision, record it in docs/DECISIONS.md, and continue. The only reason to stop is a missing docs/PRD.md.

SOURCE OF TRUTH
Read docs/PRD.md completely before writing code. It defines requirements (section 6), UX and the design system (7), Omarchy integration (8), architecture (9), packaging (10), milestones (11) and risks (12). Do not copy its content into code comments or other docs; link to the section instead. If code and PRD disagree, the PRD wins unless a decision in docs/DECISIONS.md says otherwise.

SCOPE
Milestone M0 (PRD 11.1) plus one thin vertical slice of M1: the Images tab and the pull overlay. Deliver a runnable, beautiful, well-tested skeleton, not a feature-complete app. Out of scope now: Compose, registry search, vulnerability scan, mirrors, Podman, build, publishing.

ENVIRONMENT CHECK (first)
Run go version, docker version (Docker may be absent or stopped: then use the fake engine), and detect whether Omarchy files exist on this machine. Do not hardcode library versions: resolve the latest with go get and pin them. Use only the charm.land/*/v2 modules for Bubble Tea, Lip Gloss, Bubbles, Huh and Glamour (PRD 7.3), plus Harmonica, colorprofile, a TOML library and a fuzzy matcher. For the Docker Go client, check with go list -m -versions and go doc which module path is current (github.com/moby/moby/client or github.com/docker/docker/client) and use the current official one with API version negotiation.

TASKS (do them in order; commit after each with a conventional commit message)
1. Repo foundation: go.mod (module path from git remote origin if set, else github.com/OWNER/lenger, noted in DECISIONS.md), .gitignore, MIT LICENSE, Makefile (build, run, test, lint, gallery, vhs, check), .golangci.yml, .github/workflows/ci.yml (build, vet, lint, test -race on linux amd64 and arm64), .goreleaser.yaml skeleton (no publishing), README stub, and a thin CLAUDE.md that lists build/test/lint commands and tells future agents to read docs/PRD.md first.
2. Package layout (create with doc.go files stating each package's single responsibility):
    cmd/lenger/                 entrypoint, flags (--gallery, --theme, --size, --debug, --engine)
    internal/app/               root Bubble Tea model, router, layout, overlays, job tray
    internal/ui/tokens/         design tokens, spacing grid, glyph and icon tables (Nerd Font and ASCII)
    internal/ui/theme/          ThemeSource interface; omarchy adapter; bundled themes; live reload
    internal/ui/style/          Lip Gloss style builders derived only from tokens
    internal/ui/anim/           Harmonica springs; tick scheduler that schedules nothing when idle
    internal/ui/chart/          sparkline, braille time-series, eighth-block bars, layer ribbon; treemap stub
    internal/ui/components/     tabs, statcard, list/table, modal, toast, palette, help, scrollbar, pill
    internal/ui/views/          dashboard, images (real); containers, projects, volumes, networks, system (stubs)
    internal/ui/zones/          mouse hit-testing registry
    internal/core/store/        normalized cache, indices, event-driven invalidation
    internal/core/jobs/         job queue, cancellation, progress aggregation
    internal/core/services/     image, container, project, cleanup, registry (interfaces plus minimal impls)
    internal/engine/            Engine interface; docker/ (real); fake/ (scripted fixtures and event streams)
    internal/config/            TOML load, defaults, validation (PRD Appendix A)
    internal/platform/omarchy/  theme source and setup helpers (print-only, never edits user config)
    internal/gallery/           --gallery renderer with fixtures
    docs/  vhs/  testdata/
3. Engine layer: define the Engine interface from PRD 9.2 and 9.5; implement the fake engine fully; implement the real Docker engine for list images, list containers, events stream with reconnect/backoff, and image pull with a parsed per-layer progress stream and classified errors (PRD REG-01). Everything else returns a typed ErrNotImplemented with a TODO(M#) tag.
4. Design system (PRD 7.2 to 7.8): tokens for every role in 7.4; bundled themes Lenger Midnight, Tokyo Night and Catppuccin Mocha; the Omarchy ThemeSource adapter with fallback (inspect the real Omarchy theme files on this machine if present; otherwise implement against a documented assumption and record it); gradient helper (wordmark, active-tab underline, progress fills only); eighth-block bars; braille charts; contrast test (text tokens at least 4.5:1 per theme); a lint test that fails if any file outside internal/ui/tokens and internal/ui/theme contains a hard-coded color or ad-hoc padding.
5. App shell: gradient wordmark header with daemon status chip, tab bar with sliding underline, footer with contextual key hints and job tray, help overlay, command palette (fuzzy over registered actions, shows the equivalent CLI command), toasts, modal and backdrop dimming via Lip Gloss layers, resize handling, minimum-size message, daemon-down and permission-denied screens with the exact fix, mouse support for tabs and rows, vim and arrow keys, remappable bindings.
6. Dashboard (bento grid, PRD 7.5): health card, CPU and memory braille sparklines, disk bar, needs-attention list, events feed. Real counts from the engine; chart series from the fake engine until M2.
7. Images tab (PRD 6.1 IMG-01 to IMG-04): master/detail list with status badges, sort, filter, multi-select, inspect, remove with in-use warning, layer ribbon from image history, and the pull overlay (p) with per-layer progress, speed, ETA, cancel and hide-while-running. Wire to the real engine when available, otherwise the fake.
8. Motion (PRD 7.7): selection glide, tab underline slide, count-up numbers, toast slide, overlay fade, heartbeat dot. The scheduler must run ticks only while an animation is active; honor reduce_motion, NO_MOTION and NO_COLOR.
9. Gallery: lenger --gallery renders every component and screen with fixtures, in every bundled theme, at 80x24, 120x40, 160x45 and 200x60, to stdout without Docker, and supports --theme and --size. Snapshot these frames as golden files under testdata/ with an update flag.
10. Tests (PRD 7.9, 11.2): unit tests for services and store against the fake engine; model tests that drive Update with constructed messages; full-program runs with tea.WithInput, tea.WithOutput, tea.WithWindowSize and an ASCII color profile for stable strings; layout assertions (no line wider than the terminal, overlays inside bounds); an idle test proving no ticks are scheduled; a reduce_motion test; chaos tests for event-stream drop and malformed pull JSON. VHS tapes in vhs/ for dashboard, images and pull flow (skip when vhs is not installed).
11. Docs: docs/ARCHITECTURE.md (package map and conventions only, link to PRD 9 for the rest), docs/UX.md (how to use tokens and components, link to PRD 7), docs/DECISIONS.md (every assumption and deviation, with date).

ENGINEERING RULES
- The UI never imports engine/docker; it talks to services only. Update never blocks: all I/O runs in tea.Cmd. No global mutable state. Pass context.Context everywhere and wrap errors with %w.
- Table-driven tests. Files under about 400 lines. Every exported identifier documented. Every TODO carries a milestone tag, for example TODO(M2).
- Never hard-code a color or a spacing value outside the tokens package.

SELF-VERIFICATION LOOP
After every task run make check: gofmt check, go vet, golangci-lint, go test ./... -race, the token lint, and the gallery golden check. Fix every failure and repeat until green. If a library API differs from expectation, read it with go doc -all and adapt. Never weaken a test, lint rule or gate to get green; if a gate is wrong, fix it and record why in DECISIONS.md.

DEFINITION OF DONE (all must hold)
- go build ./... succeeds and make check is green.
- lenger runs against real Docker when available and with LENGER_ENGINE=fake always.
- lenger --gallery renders all screens in all bundled themes at the four sizes with no panic and no overflow.
- The idle test passes and animations respect reduce_motion.
- docs/DECISIONS.md lists every assumption and deviation.

FINAL REPORT
Reply with: what was built, what is stubbed (by PRD requirement ID), the exact commands to run the app, the gallery and the tests, and a proposed plan for M1 as a prompt that references docs/PRD.md section 11.1.
```

After the skeleton lands, hand the next milestone to the agent with a short prompt that points to the PRD, for example: *“Implement milestone M1 as defined in docs/PRD.md section 11.1. Follow docs/ARCHITECTURE.md and docs/UX.md, run make check after each step, and fix failures until it is green.”*
