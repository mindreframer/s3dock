# Spec: Streaming S3 Download with Accurate Progress Bar for `s3dock pull`

## Purpose & User Problem

**Problem:**  
Currently, the `s3dock pull` command displays a progress bar for image downloads, but it jumps instantly from 0% to 100%. This is because the entire file is first downloaded into memory, and only then written to disk with the progress bar, making the progress bar misleading and useless for large images.

**Goal:**  
Stream the S3 download directly to disk, updating the progress bar in real time, so users see accurate download progress for large Docker images.

---

## Success Criteria

- The progress bar updates smoothly and accurately as the image is downloaded from S3, reflecting true download progress.
- Memory usage is minimal and does not scale with image size.
- The implementation works for all S3-backed downloads in the puller.
- All existing and new tests pass, including for large files.
- No regressions in error handling, retries, or checksum verification.

---

## Scope & Constraints

### In Scope

- Update the `S3Client` interface to support streaming downloads (e.g., `DownloadStream`).
- Implement streaming download in the S3 client (using AWS SDK’s `GetObject` streaming).
- Refactor `ImagePuller.downloadImageWithProgress` to use the streaming API and update the progress bar as bytes are read from S3.
- Update or add tests/mocks as needed.

### Out of Scope

- Changes to the upload/push path (this spec is only for downloads).
- Major refactoring of unrelated code.
- CLI/UI changes beyond the progress bar.

---

## Technical Considerations

- The new `DownloadStream` method should return an `io.ReadCloser` for the S3 object, allowing streaming reads.
- The progress bar should wrap the S3 stream, not a memory buffer.
- Ensure proper closing of streams and cleanup on error.
- The implementation must be compatible with the existing retry and checksum logic.
- Update mocks and tests to support the new streaming method.
- Consider backward compatibility for any other consumers of `S3Client`.

---

## Out of Scope

- Multi-part or parallel downloads.
- S3 upload progress bars.
- Support for non-S3 backends (unless already present).

---

## Questions for User

1. Is it acceptable to require all S3 clients to implement the new streaming method, or should we provide a fallback for legacy/mock clients?
2. Should we add a CLI flag to disable the progress bar, or is always-on progress fine?
3. Any specific large file sizes you want tested for this? 