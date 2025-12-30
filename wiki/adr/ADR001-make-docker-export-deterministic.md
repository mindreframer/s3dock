# ADR 001: Make Docker Export Deterministic

## Status
Accepted (2025-12-30)

## Context

When pushing Docker images to S3, s3dock exports the image using `docker save`, compresses it with gzip, and calculates an MD5 checksum for deduplication. The checksum comparison is meant to:

1. Skip re-uploading identical images (efficiency)
2. Detect content conflicts when the same tag points to different content (safety)

However, we discovered that **`docker save` produces non-deterministic output** - the same image produces different tar files on each export, even though the actual image content is identical.

### Evidence

Running `docker save` multiple times on the same image:

```bash
$ docker save myapp:tag | md5
36dadf6d63b29ac28645f3969ddb4ce3

$ docker save myapp:tag | md5  
cb9e4f72ffe79a0bd50305197de1ee0f

$ docker save myapp:tag | md5
89fa0ad47284b9b3bb877ecfbca5999b
```

### Root Cause

The tar format includes metadata timestamps that change on each export:

```bash
$ docker save myapp:tag | tar -tv | head -3
drwxr-xr-x  0 0  0  0 Dec 30 17:18 blobs/
drwxr-xr-x  0 0  0  0 Dec 30 17:33 blobs/sha256/  # <- Changes each time!
-rw-r--r--  0 0  0  482 Dec 30 17:18 blobs/sha256/...
```

The `blobs/sha256/` directory timestamp reflects when `docker save` was executed, not when the image was built.

### Current Problem

This non-determinism causes:

1. **False positive checksum mismatches**: Re-pushing the same image triggers "checksum mismatch" errors
2. **Failed archiving**: Code tries to archive the "old" file, but hits S3 key prefix bugs
3. **User frustration**: "I just want to push the same image, it should be a no-op!"

## Decision

**Normalize tar file timestamps during export to make Docker image exports deterministic.**

We will:

1. Add a `NormalizeTar()` function that rewrites all tar header timestamps to a fixed value
2. Use the **git commit timestamp** as the fixed time (already available in build context)
3. Apply normalization in the export pipeline: `docker save → normalize tar → gzip → checksum`
4. This makes checksums reproducible while maintaining full Docker compatibility

### Why this works

- Tar file content (layers, manifests, configs) is already deterministic - only metadata changes
- Setting `ModTime` to a fixed value doesn't affect Docker's ability to load the image
- The git commit timestamp is meaningful (represents when the code was committed)
- No changes to Docker itself required

### Why not alternatives?

**Alternative 1: Skip checksum comparison entirely**
- ❌ Loses ability to detect actual content conflicts
- ❌ Can't distinguish between "same image" and "different image with same tag"

**Alternative 2: Compare by tag only (if metadata exists, skip)**
- ❌ Can't detect if someone rebuilds the same tag with different content
- ❌ No guarantee that git hash corresponds to actual image content

**Alternative 3: Use image digest instead of export**
- ❌ Docker digest is for registry manifest, not tar export
- ❌ Doesn't help with tar export deduplication

**Our approach:**
- ✅ Preserves checksum-based deduplication
- ✅ Detects actual content conflicts
- ✅ Maintains full Docker compatibility
- ✅ Uses meaningful timestamp (git commit time)

## Implementation

### Critical Detail: Gzip Timestamp

In addition to tar normalization, **gzip compression must also be deterministic**. By default, gzip includes a timestamp in its header which changes on each compression. This is fixed by setting `ModTime` to zero time:

```go
gzipWriter := gzip.NewWriter(output)
gzipWriter.ModTime = time.Time{} // Critical for deterministic output
```

Without this, even with normalized tar, the final checksum would still vary.

### Core Function

```go
// NormalizeTar reads a tar stream and rewrites all timestamps to fixedTime
func NormalizeTar(input io.Reader, output io.Writer, fixedTime time.Time) error {
    tarReader := tar.NewReader(input)
    tarWriter := tar.NewWriter(output)
    defer tarWriter.Close()

    for {
        header, err := tarReader.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        // Normalize ModTime (AccessTime/ChangeTime not supported in USTAR)
        header.ModTime = fixedTime
        header.AccessTime = time.Time{}
        header.ChangeTime = time.Time{}

        if err := tarWriter.WriteHeader(header); err != nil {
            return err
        }

        if header.Typeflag == tar.TypeReg {
            if _, err := io.Copy(tarWriter, tarReader); err != nil {
                return err
            }
        }
    }

    return nil
}
```

### Integration Points

**In `internal/pusher.go`:**

```go
// Export Docker image
imageData, err := p.docker.ExportImage(ctx, imageRef)

// Normalize tar timestamps before compression
pr, pw := io.Pipe()
go func() {
    defer pw.Close()
    if err := NormalizeTar(imageData, pw, gitTime); err != nil {
        pw.CloseWithError(err)
    }
}()

// Continue with gzip compression and checksum calculation
gzipWriter := gzip.NewWriter(...)
```

## Consequences

### Positive

- ✅ **Deterministic exports**: Same image = same checksum every time
- ✅ **Reliable deduplication**: Skip uploads work as designed
- ✅ **Conflict detection**: Different content = different checksum
- ✅ **Meaningful timestamps**: All files show git commit time
- ✅ **No Docker changes**: Pure tar metadata manipulation
- ✅ **Fully reversible**: Can load normalized tar back into Docker

### Negative

- ⚠️ **Slight performance overhead**: Additional tar read/write pass
  - *Mitigation*: Streaming architecture, no disk I/O, minimal CPU
- ⚠️ **Timestamp semantics**: ModTime shows commit time, not build time
  - *Mitigation*: This is actually more correct - commit time is the meaningful timestamp

### Neutral

- File sizes remain identical (only metadata changes)
- Docker layer caching unaffected
- Image functionality unchanged

## Validation

### Proof of Concept Results

POC script (`poc_deterministic_tar.go`) demonstrates:

```bash
$ go run poc_deterministic_tar.go myapp:tag /tmp/test1.tar.gz
MD5 checksum: 050d413c79296002736a3894609781f1

$ go run poc_deterministic_tar.go myapp:tag /tmp/test2.tar.gz
MD5 checksum: 050d413c79296002736a3894609781f1

$ go run poc_deterministic_tar.go myapp:tag /tmp/test3.tar.gz
MD5 checksum: 050d413c79296002736a3894609781f1
```

✅ **Identical checksums across multiple runs**

### Docker Compatibility

```bash
$ gunzip -c /tmp/test1.tar.gz | docker load
Loaded image: myapp:tag
```

✅ **Normalized tar loads successfully into Docker**

## References

- [Reproducible Builds](https://reproducible-builds.org/)
- [tar format specification](https://www.gnu.org/software/tar/manual/html_node/Standard.html)
- [Docker save documentation](https://docs.docker.com/engine/reference/commandline/save/)

## Notes

The git commit timestamp is already parsed in `pusher.go` for tagging purposes. We use `time.Parse()` to convert the git timestamp string (format: `20251230-1718`) to a `time.Time` value for tar normalization.
