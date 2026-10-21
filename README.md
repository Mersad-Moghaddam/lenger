# Lenger

**Anchor your containers.** A container-first Docker terminal UI for Linux, with
optional Omarchy desktop integration and macOS binary builds.

[![Go Version](https://img.shields.io/badge/go-1.27+-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## The cockpit

Four resource views keep navigation focused: **Containers**, **Images**,
**Volumes**, and **Networks**. Containers open a five-tab inspector:
**Health & Diagnostics**, **Live Logs**, **Metrics**, **Env & Ports**, and
**Timeline**. Health evidence, sampled telemetry, and diagnostic hypotheses
are labeled separately; unavailable data must not be read as a healthy result.

Use the offline fixture mode to explore without contacting Docker. Fake-engine
behavior and gallery snapshots are not proof that a real-daemon operation works.
Validate against a disposable daemon before using destructive operations on
important workloads. See [distribution and verification](docs/DISTRIBUTION.md).

## Installation

### GitHub Releases

Release archives target Linux (`amd64`, `arm64`, `armv7`) and macOS (`amd64`,
`arm64`). The following example selects **v0.2.0**, not a moving `latest` URL.
Use it only once that version and its assets are present on
[Releases](https://github.com/Mersad-Moghaddam/lenger/releases).

Linux amd64, with `curl`, `sha256sum`, `tar`, and `install` available:

```sh
(
  set -eu
  version=0.2.0
  archive="lenger_${version}_linux_amd64.tar.gz"
  base="https://github.com/Mersad-Moghaddam/lenger/releases/download/v${version}"
  workdir="$(mktemp -d)"
  trap 'rm -rf "$workdir"' EXIT
  cd "$workdir"
  curl --fail --location --show-error --output "$archive" "$base/$archive"
  curl --fail --location --show-error --output SHA256SUMS "$base/SHA256SUMS"
  awk -v name="$archive" '$2 == name { print; found++ } END { if (found != 1) exit 1 }' SHA256SUMS > selected.sha256
  sha256sum --check selected.sha256
  tar -xzf "$archive"
  ./lenger --version
  sudo install -m 755 lenger /usr/local/bin/lenger
)
```

Change the archive suffix for other targets. On macOS use a compatible SHA-256
verification tool (for example `shasum -a 256 --check selected.sha256`). Do not
skip verification. Unsigned checksums downloaded from the same release detect
corruption; they do **not** independently authenticate the publisher.

### From source

Requires Go 1.27+ and Make:

```sh
git clone https://github.com/Mersad-Moghaddam/lenger.git
cd lenger
make build
./bin/lenger --version
sudo install -m 755 bin/lenger /usr/local/bin/lenger
```

Source builds default to version `dev`; approved version builds can use
`make build VERSION=0.2.0 COMMIT="$(git rev-parse HEAD)" BUILD_DATE="$(git show -s --format=%cI HEAD)"`.
Only assign a release version to its reviewed source commit.

### Arch Linux / AUR

`packaging/aur` and `packaging/aur-bin` are **unpublished templates**, not an
installation channel for this release. Their placeholder `SKIP` checksums must be
replaced with independently verified source/archive hashes and clean package
builds before a separate AUR publication. No AUR publication is claimed here.

## Usage

```sh
lenger --version                   # offline; no Docker/config/TUI initialization
lenger --help
lenger                             # real Docker engine
lenger --theme midnight            # bundled palette
lenger --theme omarchy             # optional desktop palette source
lenger --engine fake --theme midnight  # offline interactive fixture demo
lenger --gallery --theme midnight --size 120x40  # offline text gallery
```

Docker mode requires access to the intended daemon. Check `DOCKER_HOST` before
launching; named Docker contexts are not supported. Do not point integration
tests at production containers. Optional operations may depend on external tools
such as Docker Compose or vulnerability scanners; failures are not a successful
result.

## Keyboard reference

| Key | Action |
| --- | --- |
| `1` / `2` / `3` / `4` | Containers / Images / Volumes / Networks |
| `j` / `k` or arrows | Move through resources |
| `/` | Filter current resource list |
| `Enter` | Inspect selected resource; open container drawer |
| `Tab` / `Shift+Tab` | Move focus in the cockpit |
| `:` / `Ctrl+P` | Command and resource palette |
| `?` | Contextual help |
| `R` | Refresh resources |
| `Esc` | Close current context or clear filter |
| `q` | Quit / close current overlay |

Docker log records retain up to 64 KiB each (including the timestamp prefix);
larger records retain a prefix with an explicit `[truncated]` marker. Live and
previous-log views retain at most 10,000 records and 16 MiB of log text, evicting
oldest records when either limit is reached. These are text-storage limits, not
total process memory limits: entry indexes, stream metadata, transport buffers
and transient viewport/copy snapshots add overhead. Ordinary records within the
cap keep their full diagnostic content for search, level filtering and copying.

The container drawer has five inspectors: Health & Diagnostics, Live Logs,
Metrics, Env & Ports, and Timeline. Follow the visible drawer hints for
tab selection and log follow/filter controls; keys are contextual.
Image actions and auxiliary workflows are discoverable through the command
palette and help rather than additional numbered workspaces. Read confirmation
previews carefully before any stop, restart, or removal.

## Configuration

Configuration is read from `$XDG_CONFIG_HOME/lenger/config.toml`, normally
`~/.config/lenger/config.toml`. A missing file uses defaults. Example:

```toml
[ui]
theme = "midnight"
ascii_icons = false
reduce_motion = false

[docker]
safety_poll_seconds = 30

[logs]
buffer_lines = 10000
show_timestamps = false
```

Use `NO_COLOR` for colorless output and `NO_MOTION=1` to reduce motion.
Never store registry credentials or other secrets in this file.

## Optional Omarchy setup

```sh
lenger setup-omarchy             # preview only; no files changed
lenger setup-omarchy --apply     # explicitly request a bindings-file append
```

Review the proposed terminal, config path, and `Super+Alt+D` binding first.
Automatic setup is a best-effort append to an existing detected config, not a
complete conflict detector or transactional desktop configurator. Back up your
bindings and inspect the command's output; if append fails, apply the printed
binding manually. It does not create a missing config or reload Hyprland for you.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for tests and review expectations,
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for boundaries, and
[docs/DISTRIBUTION.md](docs/DISTRIBUTION.md) for release gates. CI results apply
only to the tested commit, not automatically to a later tag or working tree.

## License

MIT © [Mersad Moghaddam](https://github.com/Mersad-Moghaddam)
