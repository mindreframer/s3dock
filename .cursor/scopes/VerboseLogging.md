# Verbose Logging Implementation Spec

## Purpose & User Problem

Users need better visibility into s3dock operations for debugging, troubleshooting, and understanding what's happening during complex workflows. Currently, s3dock provides minimal output, making it difficult to diagnose issues or understand the flow of operations.

## Success Criteria

1. **CLI Flag Integration**: `--log-level` (or `-l`) accepts values 1, 2, 3
2. **Log Levels**:
   - Level 1: Errors only (critical failures)
   - Level 2: Info + Errors (default, normal operations)
   - Level 3: Debug + Info + Errors (detailed trace)
3. **Consistent Logging**: All commands respect the log level
4. **Clean Output**: Timestamped, structured logs that are easy to parse
5. **No Breaking Changes**: Existing functionality and output remain intact

## Scope & Constraints

### In Scope
- Add `--log-level` flag to all s3dock commands
- Implement logging interface in `internal/log.go`
- Replace all existing `fmt.Println`, `log.Print` with structured logging
- Log all major operations: build, push, tag, promote, pull, deploy
- Include relevant context in debug logs (file paths, S3 keys, Docker image IDs)
- Add integration tests for log level behavior

### Out of Scope
- File logging (future enhancement)
- Colorized output (future enhancement)
- Environment variable overrides (future enhancement)
- External logging libraries (use stdlib)

## Technical Considerations

### Logging Interface Design
```go
type Logger interface {
    Error(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Debug(msg string, args ...interface{})
    SetLevel(level int)
}
```

### Log Level Implementation
- Level 1: Only Error() calls output
- Level 2: Error() and Info() calls output  
- Level 3: Error(), Info(), and Debug() calls output

### Integration Points
- **main.go**: Add flag parsing and logger initialization
- **internal/**: Replace all print statements with logger calls
- **CLI commands**: Ensure all commands use the logger
- **Error handling**: Maintain existing error return patterns

### Log Message Examples
```
[ERROR] 2025-01-27 10:30:15 Failed to upload image to S3: access denied
[INFO]  2025-01-27 10:30:15 Building image myapp with tag 20250127-1030-a1b2c3d
[DEBUG] 2025-01-27 10:30:15 Docker build context: ./backend
[DEBUG] 2025-01-27 10:30:15 S3 upload path: images/myapp/202501/myapp-20250127-1030-a1b2c3d.tar.gz
```

## Implementation Plan

### Phase 1: Core Logging Infrastructure
1. Create `internal/log.go` with Logger interface and implementation
2. Add global logger instance and level management
3. Add `--log-level` flag to root command in `main.go`

### Phase 2: Replace Existing Logging
1. Update `internal/builder.go` - build operations
2. Update `internal/pusher.go` - push operations  
3. Update `internal/tagger.go` - tag operations
4. Update `internal/pointer.go` - promote operations
5. Update `internal/docker.go` - Docker operations
6. Update `internal/s3.go` - S3 operations
7. Update `internal/git.go` - Git operations

### Phase 3: Testing & Validation
1. Add integration tests for log level behavior
2. Test all commands with different log levels
3. Verify no sensitive data leaks in debug logs
4. Ensure existing functionality unchanged

### Phase 4: Documentation
1. Update `Readme.md` with logging examples
2. Update CLI help text
3. Add logging examples to documentation

## Risk Mitigation

- **Performance**: Logger calls should be minimal overhead
- **Sensitive Data**: Ensure no credentials or secrets in debug logs
- **Backward Compatibility**: Default log level 2 maintains current behavior
- **Error Handling**: Logger failures shouldn't break main functionality

## Acceptance Criteria

1. `s3dock --log-level 1 push myapp` shows only errors
2. `s3dock --log-level 2 push myapp` shows info + errors (default)
3. `s3dock --log-level 3 push myapp` shows debug + info + errors
4. All existing commands work unchanged with default behavior
5. Integration tests pass for all log levels
6. No sensitive data appears in any log level 