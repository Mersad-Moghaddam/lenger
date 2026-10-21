# Public distribution roadmap

Repository: https://github.com/Mersad-Moghaddam/lenger

Publishing source is not the same as publishing a stable installable release.
The current code is a pre-release skeleton, with no verified binary artifacts.

## Appropriate channels

- **GitHub Releases:** primary home for Linux amd64/arm64 binaries, source,
  checksums, versioned release notes and installation instructions.
- **Arch User Repository:** `lenger` (source) and later `lenger-bin` (releases),
  after checking name availability and validating builds in a clean environment.
- **Terminal Trove / relevant awesome lists:** discovery once a working demo and
  accurate feature list exist; not package hosting.
- **Omarchy community resources:** follow the current documented submission
  process rather than assuming a universal Omarchy marketplace exists.

GitHub Marketplace distributes GitHub Apps and Actions. Lenger is neither, so
it should not be submitted there as a standalone desktop/TUI application.

## Gates before the first tagged release

1. Resolve dependency checksums and transitive requirements; commit `go.sum`.
2. Build, vet, lint, race-test and review all results on both Linux architectures.
3. Test against disposable real Docker daemons, including permissions,
   interrupted streams, cancellation and non-forced deletion conflicts.
4. Render/review the gallery, commit honest baselines, and record real demos.
5. Finish or explicitly defer incomplete requirements; do not advertise stubs.
6. Review the credential helper path and source for secrets and unsafe defaults.
7. Enable automatic CI triggers; pin release tooling/actions after validation.
8. Enable release publishing only for an approved tag. Never publish placeholder
   artifacts or checksums for binaries that were not actually built.

No release, AUR package, or external listing was published during the coding-only
review. Those are separate operations with their own validation requirements.
