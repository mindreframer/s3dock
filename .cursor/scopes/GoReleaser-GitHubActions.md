# Spec: Automated Binary Releases with GoReleaser and GitHub Actions

**Purpose & User Problem**  
Enable automated, reproducible, and cross-platform binary releases of s3dock, distributed via GitHub Releases. Remove manual steps, ensure users can easily download pre-built binaries for their OS/arch, and streamline the release process.

---

## Success Criteria

- On pushing a new git tag (e.g., v1.2.3), GitHub Actions builds s3dock for major OS/architectures (Linux, macOS, Windows; amd64/arm64).
- Binaries are uploaded as assets to the corresponding GitHub Release.
- Release includes checksums (SHA256) for all binaries.
- (Optional) Release includes SBOM and/or signature files.
- Release process is fully automated—no manual upload steps.
- GoReleaser config is versioned in the repo.
- Documentation exists for maintainers on how to trigger a release.

---

## Scope & Constraints

**In Scope:**
- GoReleaser config (.goreleaser.yaml) for s3dock
- GitHub Actions workflow for GoReleaser
- Multi-platform builds (at least: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- Checksums generation and upload
- Release notes auto-generated from git history

**Out of Scope:**
- Docker image publishing (unless requested)
- Homebrew or other package manager integration
- S3 publishing (for binaries) via GoReleaser
- Advanced signing (unless requested)

---

## Technical Considerations

- GoReleaser requires a clean git state and a valid Go module.
- GitHub Actions must have permissions to create releases and upload assets.
- GoReleaser can be run in “snapshot” mode locally for testing.
- Binaries should be named with OS/arch suffixes for clarity.
- Release notes can be auto-generated or customized.

---

## Out of Scope Items

- Custom installers (e.g., MSI, DMG)
- Announcements to Slack, Discord, etc.
- Non-GitHub release platforms

---

## Questions for You

1. Do you want to include SBOM (Software Bill of Materials) or signature files in the release?
2. Do you want to customize the binary naming or stick with defaults?
3. Should we include a “snapshot” release workflow for testing (does not publish to GitHub)?
4. Any other platforms/architectures needed? 