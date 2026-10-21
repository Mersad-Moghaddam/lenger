# Design Specification: Milestone M3 — Registry Power & Image Mastery

**Author:** Lenger Architecture Team  
**Date:** 2026-10-21  
**Status:** Approved  
**Target Milestone:** M3 (PRD.md Section 11.1)

---

## 1. Intent & Developer Friction

As developers using Docker day-to-day, five points of friction create constant friction:
1. **Blind Pulls and Silent Stale Tags (IMG-09):** Developers run `latest` or floating version tags without knowing if remote upstream has security fixes or newer layers. Lenger makes digest comparison instant: pressing `U` queries the remote registry manifest digest without pulling the image, displaying an `UPDATE AVAILABLE` badge and offering single-key pull-and-recreate.
2. **Mirror Retries in Restricted or High-Latency Networks (REG-02, REG-03):** When Docker Hub or GHCR times out or hits 429/403 rate limits, developers manually re-type mirror URLs, re-pull, and re-tag. Lenger detects registry blocks, offers one-key retry via configured mirror mirrors, transparently rewrites the manifest path, and re-tags back to the canonical reference. It also provides a guided diff preview for `/etc/docker/daemon.json` `registry-mirrors`.
3. **Bloated Layers and Opaque History (IMG-07):** Images balloon to gigabytes without visibility into what command generated the bulk. Lenger's enhanced inspection displays a visual layer hierarchy with horizontal bar proportions and highlights the single largest layer.
4. **Vulnerability Blindness (IMG-10):** Developers don't know if base images contain critical CVEs. Lenger includes a zero-overhead scanner adapter that auto-detects `trivy`, `grype`, or `docker scout` in PATH and displays Critical / High / Medium / Low severity pills directly in the image inspector.
5. **Reverse Dependencies (IMG-11):** "Why is this image here?" Before deleting an image, developers need to know which containers, Compose projects, and parent/child layers rely on it.

---

## 2. Goals, Non-Goals, and Anti-Goals

### Goals
- **Registry Search & Tag Browser (IMG-06):** Search Docker Hub, GHCR, and Quay with official badge, star count, and remote tag enumeration in a dedicated overlay (`S`).
- **Layer Breakdown & Biggest-Layer Sounding (IMG-07):** Enhance `InspectImage` to calculate relative layer sizes, highlight the biggest layer, and format commands cleanly.
- **Tag, Retag, Push, Save, and Load (IMG-08):** Add `t` (tag/retag), `P` (push), and export/import tar workflows (`docker save` / `docker load`).
- **Update Check Engine (IMG-09):** Check remote digests via OCI Distribution API HEAD/GET manifest requests without pulling full layers.
- **Security Scanner Adapter (IMG-10):** Pluggable CLI scanner adapter parsing JSON reports from Trivy, Grype, or Docker Scout.
- **Reverse Dependency Sounding (IMG-11):** Inspect tab showing all dependent containers, Compose projects, and dependent images.
- **Mirror-Aware Pull & Daemon Config Diff (REG-02, REG-03):** Config-driven mirror fallback, mirror pull rewriting, and guided `/etc/docker/daemon.json` configuration generator.
- **Registry Auth (REG-06):** Docker credential store integration for `docker login` / `docker logout` status.

### Non-Goals
- Embedding an entire CVE database (e.g. 500MB database) inside Lenger; we delegate to installed scanners (Trivy, Grype, Scout) or display "scanner not found in PATH".
- Building a full OCI registry server inside Lenger.

### Anti-Goals
- Storing plain-text registry passwords in Lenger's config. Lenger strictly delegates to standard Docker credential helpers (`pass`, `secretservice`, `osxkeychain`, `wincred`).
- Overwriting `/etc/docker/daemon.json` silently without root permission check and explicit diff confirmation.

---

## 3. Architecture & Data Contracts

### 3.1 Registry Client & Engine Contracts (`internal/engine/`)
```go
type RegistrySearchResult struct {
    Name        string
    Description string
    Stars       int
    Official    bool
    Automated   bool
}

type RemoteTag struct {
    Name      string
    Digest    string
    Size      int64
    UpdatedAt time.Time
}

type VulnerabilityReport struct {
    Scanner   string // trivy, grype, scout
    Critical  int
    High      int
    Medium    int
    Low       int
    Fixable   int
    ScannedAt time.Time
    Error     string
}

type ImageUpdateStatus struct {
    LocalDigest    string
    RemoteDigest   string
    HasUpdate      bool
    CheckedAt      time.Time
}
```

### 3.2 Engine Methods
```go
type Engine interface {
    // ... existing M0-M2 methods ...
    SearchRegistry(ctx context.Context, query string) ([]RegistrySearchResult, error)
    ListRemoteTags(ctx context.Context, repo string) ([]RemoteTag, error)
    CheckImageUpdate(ctx context.Context, ref string, localDigest string) (ImageUpdateStatus, error)
    TagImage(ctx context.Context, sourceID, targetRef string) error
    PushImage(ctx context.Context, ref string, emit func(Progress) error) error
    SaveImage(ctx context.Context, id, targetFile string) error
    LoadImage(ctx context.Context, sourceFile string) error
}
```

### 3.3 Core Service & Scanner Adapters (`internal/core/services/`)
- `scanner.go`: Detects installed scanner (`trivy image --format json`, `grype -o json`, `docker scout cves --format json`). Parses CVE severities into `VulnerabilityReport`.
- `mirrors.go`: Manages mirror fallbacks and daemon.json diff generation.
- `registry.go`: OCI manifest digest resolution and search queries.

### 3.4 UI Overlays & View Polishing (`internal/ui/views/`, `internal/app/`)
- Image Details Pane: Tabs for Config, Layers (with biggest-layer bar chart), Dependencies ("Why is this image here?"), and Security (CVE counts).
- Overlays:
  - `registry-search`: Query registry, browse results, view tags, pull selected tag.
  - `tag-image`: Source image, new tag input, CLI preview (`docker tag`).
  - `daemon-mirrors`: Diff preview for `/etc/docker/daemon.json` with copyable sudo command.
  - `export-image` / `import-image`: Path input for `.tar` archives.

---

## 4. Acceptance Criteria
1. Registry search returns results with star counts and official badges.
2. Layer breakdown in image detail highlights largest layer and shows relative size bars.
3. "Why is this image here?" tab accurately lists containers and Compose projects using the image.
4. Update check flags image with `UPDATE AVAILABLE` when local digest differs from remote manifest.
5. Vulnerability scan gracefully parses scanner output or displays helpful install instructions when no scanner is in PATH.
6. Configurable mirrors allow fallback pull and mirror rewrite.
7. Unit tests verify all M3 services, models, views, and scanner parsers.
