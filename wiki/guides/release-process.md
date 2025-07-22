# s3dock Release Process Guide

This guide explains how to publish a new release of s3dock using GoReleaser and GitHub Actions. The process is fully automated—just tag and push!

---

## 1. Prerequisites

- Ensure all code is committed and pushed to the main branch.
- You have push access to the repository.
- Your working directory is clean (no uncommitted changes).

---

## 2. Create a New Release Tag

Choose a new version number (e.g., v1.2.3) following semantic versioning.

```
git tag v1.2.3
git push origin v1.2.3
```

---

## 3. What Happens Next?

- GitHub Actions will detect the new tag and start the release workflow.
- GoReleaser will:
  - Build s3dock for Linux, macOS, and Windows (amd64/arm64)
  - Package binaries into tar.gz archives
  - Generate a checksums.txt file
  - Create a GitHub Release and upload all binaries and checksums
  - Auto-generate release notes from commit history

---

## 4. Downloading Binaries

- Visit the [GitHub Releases page](https://github.com/mindreframer/s3dock/releases)
- Download the appropriate binary for your OS/architecture

---

## 5. Troubleshooting

- If the release fails, check the Actions tab for logs and errors.
- Ensure your tag matches the `v*` pattern (e.g., v1.2.3).
- The workflow requires a clean git state and a valid Go module.

---

## 6. Advanced

- To test the release process locally (without publishing), you can run:
  ```
  goreleaser release --snapshot --clean
  ```
- For more info, see [GoReleaser documentation](https://goreleaser.com/)

---

Happy releasing! 