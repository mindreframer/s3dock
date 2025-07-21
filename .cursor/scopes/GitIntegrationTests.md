# Spec: Git Client Integration Tests

## Purpose & User Problem

Ensure that the s3dock Git client logic (as used in `s3dock build`) works correctly in real-world scenarios, including interaction with Docker builds. This ensures reproducible, traceable builds and prevents deployment surprises due to git state issues.

## Success Criteria
- Tests run in isolated, temporary git repos with minimal Dockerfiles.
- Tests verify correct behavior for clean and dirty git states.
- Tests verify correct tag generation and build success/failure as appropriate.
- Tests can be run in parallel and clean up after themselves unless `--keep-tmp` is set.
- Tests support verbose logging via `--log-level` argument.
- Tests are executed via `make test-integration` and not as part of the default unit test suite.

## Scope & Constraints
- Only covers git operations relevant to `s3dock build` (e.g., clean/dirty state, commit hash/timestamp extraction, tag format).
- Each test creates its own temp directory and git repo.
- Minimal placeholder Dockerfile is used (e.g., `FROM busybox`).
- Tests interact with Docker to ensure build logic works end-to-end.
- Tests assume Docker is running and available.
- Tests are written in Go and use Go functions directly (not CLI subprocesses).
- Temp directories are cleaned up unless `--keep-tmp` is provided.
- Logging verbosity is controlled via `--log-level` CLI argument (levels 1, 2, 3).

## Technical Considerations
- Use Go's `testing` package and `t.Parallel()` for parallelism.
- Use `os.MkdirTemp` for temp directories.
- Use `os/exec` to run git and docker commands as needed.
- Provide helpers for repo setup, commit, dirty state, etc.
- Provide helpers for logging and temp dir management.
- Tests should fail if Docker is not available.
- Integration tests should be placed in a file only run by `make test-integration`.

## Out of Scope
- No tests for other s3dock commands (push, tag, promote, etc.) at this stage.
- No tests for advanced git scenarios (e.g., submodules, large repos, merge conflicts).
- No tests for S3 interactions or networked storage.
- No tests for Docker build edge cases (focus is on git integration).

## Example Test Cases
- Clean repo, single commit, build succeeds, tag is correct.
- Dirty repo, build fails as expected.
- Commit, then modify Dockerfile, check dirty detection.
- (Optional) Multiple commits, check tag format.
