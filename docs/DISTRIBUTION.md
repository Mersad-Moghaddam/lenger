# Distribution and release verification

Repository: https://github.com/Mersad-Moghaddam/lenger

## Supported publication path

**GitHub Releases is the only publication channel in this release task.**
`.github/workflows/release.yml` is authoritative. The unused GoReleaser config
has been retired so it cannot silently produce a different archive/checksum
contract. A workflow definition or local build is not proof of publication.

Stable tags must be `vMAJOR.MINOR.PATCH` with no leading zeros, prerelease suffix,
or build suffix. The release workflow checks the tag resolves to the triggering
commit and calls the same verification workflow used by CI, checking out that
exact SHA on both Linux amd64 and arm64 runners. Publishing depends on success
of that verification job, not merely a previous main-branch check.

Verification runs formatting checks, build, vet, lint, race tests, and gallery
goldens through `make check`. The subsequent release job cross-builds five
CGO-free targets:

- `lenger_VERSION_linux_amd64.tar.gz`
- `lenger_VERSION_linux_arm64.tar.gz`
- `lenger_VERSION_linux_armv7.tar.gz`
- `lenger_VERSION_darwin_amd64.tar.gz`
- `lenger_VERSION_darwin_arm64.tar.gz`

Each archive contains `lenger`, `lenger.desktop`, `README.md`, and `LICENSE` at
its root. `SHA256SUMS` covers all five archives. Version, triggering commit, and
source commit date are injected into the binaries and available via `--version`
without config, Docker, or a terminal. The README demonstrates download,
checksum verification, and installation in that order. Same-release unsigned
checksums provide integrity, not independent publisher authentication.

## Maintainer release checklist

1. Finish owner changes and strict review. Reconcile public usage docs with the
   actual four-resource / five-inspector UI; defer unsupported claims.
2. Run `make check` on the final committed tree, inspect gallery output, and
   review credential handling and destructive-operation safeguards. Record
   explicitly authorized disposable-daemon integration coverage and its gaps.
3. Build all five targets, inspect archive layout and checksums, and smoke-test
   host `--version`, help, and offline gallery. Cross-compilation alone does not
   validate execution on another operating system/architecture.
4. Push the reviewed commit and wait for green exact-commit CI. Only the parent
   release operator may create/push the approved stable tag. The tag workflow
   reruns verification before build/publication; confirm both runner types are
   available and do not bypass failures.
5. Inspect the resulting GitHub release: correct tag/commit, all five assets,
   `SHA256SUMS`, downloaded checksum validation, archive layout, and reported
   version. Do not claim publication or remote artifact verification until done.

Actions currently use upstream major-version tags and lint currently resolves
`latest`. This is a tooling reproducibility limitation, not an immutable supply
chain. Pin a linter only after verifying its Go 1.27 compatibility, then pin
reviewed upstream action commits and arrange dependency updates; do not invent
SHAs or a supposedly compatible linter version. Runner/tool incompatibility must
fail the release gate, never be waived as an assumed pass.

## AUR is deferred

`packaging/aur/PKGBUILD` and `packaging/aur-bin/PKGBUILD` are **unpublished
v0.2.0 templates**. `SKIP` entries are unresolved placeholders, not approved
integrity policy. The source template installs its desktop file from the
extracted source tree (`packaging/lenger.desktop`), not a nonexistent local AUR
source. Before any separately authorized AUR publication:

- Confirm package name availability and maintainer ownership.
- Download the actual approved tagged source and binary archives, independently
  verify them, and replace every `SKIP` with a real SHA-256 digest.
- Generate `.SRCINFO` from finalized recipes and review actual upload contents.
- Build/install in clean Arch environments for supported architectures and
  verify desktop metadata, licenses, and `lenger --version`.

No AUR listing, package ownership, clean Arch build, or publication is claimed.
Do not instruct users to install these templates as a verified AUR channel.

## Other channels

Terminal Trove and relevant awesome lists are discovery channels, not package
hosting. Follow current Omarchy community submission guidance rather than
assuming a universal marketplace. GitHub Marketplace distributes Apps and
Actions; Lenger is neither.
