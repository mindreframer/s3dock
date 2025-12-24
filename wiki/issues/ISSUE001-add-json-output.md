# ISSUE001: Add JSON Output Support for Programmatic Consumption

## Problem

s3dock currently outputs all results as human-readable text to stdout. This makes it fragile to use programmatically, as consumers must parse unstructured string output which can break when formatting changes.

## Goal

Add a `--json` global flag that makes all commands emit structured JSON output for reliable programmatic consumption.

---

## Current State Analysis

### Output locations that need modification

| Command | File | Current Output | Data Available |
|---------|------|----------------|----------------|
| `build` | `main.go` + `internal/builder.go` | Logs only, returns image tag | Image tag string |
| `push` | `main.go` + `internal/pusher.go` | Logs only | S3 key, checksum, skipped flag |
| `tag` | `main.go` + `internal/tagger.go` | Logs only | Image ref, version, S3 key |
| `promote` | `main.go` + `internal/tagger.go` | Logs only | Source, environment, skipped flag |
| `pull` | `main.go` + `internal/puller.go` | Logs only | Image ref, source |
| `current` | `main.go` (line 753) | `fmt.Println(imageRef)` | Image reference |
| `list apps` | `main.go` (lines 846-851) | `fmt.Println(app)` per line | `[]string` |
| `list images` | `main.go` (lines 897-902) | `fmt.Printf("%s:%s\n")` per line | `[]ImageInfo` struct |
| `list tags` | `main.go` (lines 939-944) | `fmt.Printf("%s -> %s\n")` per line | `[]TagInfo` struct |
| `list envs` | `main.go` (lines 981-989) | `fmt.Printf(...)` per line | `[]EnvInfo` struct |
| `list tag-for` | `main.go` | Single line output | Tag string or empty |
| `version` | `main.go` (lines 769-773) | Multiple lines or single | Version, commit, date |
| `config show` | `main.go` (lines 236-248) | `fmt.Printf` formatted | Profile struct |
| `config list` | `main.go` (lines 258-264) | Formatted list | Profile names |

### Existing good patterns

- `ImageInfo`, `TagInfo`, `EnvInfo` structs in `internal/list.go` already have JSON tags
- `PointerMetadata` has JSON serialization (`ToJSON()`)
- `ImageMetadata` has JSON serialization

---

## Implementation Plan

### Phase 1: Core Infrastructure

1. **Add output format enum and global flag** (`internal/output.go` - new file)
   ```go
   type OutputFormat int
   const (
       OutputFormatText OutputFormat = iota
       OutputFormatJSON
   )
   
   type OutputConfig struct {
       Format OutputFormat
   }
   
   var globalOutputConfig = &OutputConfig{Format: OutputFormatText}
   ```

2. **Add `--json` global flag** (`main.go`)
   - Add to `GlobalFlags` struct: `JSON bool`
   - Parse in `parseGlobalFlags()`
   - Set `globalOutputConfig` before command execution

3. **Create result types** (`internal/results.go` - new file)
   ```go
   // Generic wrapper for all command results
   type CommandResult struct {
       Success bool        `json:"success"`
       Command string      `json:"command"`
       Data    interface{} `json:"data,omitempty"`
       Error   string      `json:"error,omitempty"`
   }
   
   // Specific result types
   type BuildResult struct {
       ImageTag string `json:"image_tag"`
       AppName  string `json:"app_name"`
       GitHash  string `json:"git_hash"`
       GitTime  string `json:"git_time"`
   }
   
   type PushResult struct {
       ImageRef string `json:"image_ref"`
       S3Key    string `json:"s3_key"`
       Checksum string `json:"checksum"`
       Size     int64  `json:"size"`
       Skipped  bool   `json:"skipped"`
       Archived bool   `json:"archived"`
   }
   
   type TagResult struct {
       ImageRef string `json:"image_ref"`
       Version  string `json:"version"`
       S3Key    string `json:"s3_key"`
   }
   
   type PromoteResult struct {
       Source      string `json:"source"`
       Environment string `json:"environment"`
       SourceType  string `json:"source_type"` // "image" or "tag"
       Skipped     bool   `json:"skipped"`
   }
   
   type PullResult struct {
       ImageRef    string `json:"image_ref"`
       Source      string `json:"source"`
       SourceType  string `json:"source_type"` // "environment" or "tag"
       Skipped     bool   `json:"skipped"`
   }
   
   type CurrentResult struct {
       AppName     string `json:"app_name"`
       Environment string `json:"environment"`
       ImageRef    string `json:"image_ref"`
   }
   
   type ListAppsResult struct {
       Apps []string `json:"apps"`
   }
   
   type ListImagesResult struct {
       AppName string      `json:"app_name"`
       Images  []ImageInfo `json:"images"`
   }
   
   type ListTagsResult struct {
       AppName string    `json:"app_name"`
       Tags    []TagInfo `json:"tags"`
   }
   
   type ListEnvsResult struct {
       AppName      string    `json:"app_name"`
       Environments []EnvInfo `json:"environments"`
   }
   
   type VersionResult struct {
       Version string `json:"version"`
       Commit  string `json:"commit"`
       Date    string `json:"date"`
   }
   ```

4. **Create output helper function** (`internal/output.go`)
   ```go
   func OutputResult(result interface{}) error {
       if globalOutputConfig.Format == OutputFormatJSON {
           return outputJSON(result)
       }
       return outputText(result)
   }
   
   func outputJSON(result interface{}) error {
       encoder := json.NewEncoder(os.Stdout)
       encoder.SetIndent("", "  ")
       return encoder.Encode(result)
   }
   ```

### Phase 2: Modify Internal Functions to Return Results

Currently, internal functions (e.g., `Push()`, `Build()`) only return errors and log to stderr. They need to return result structs.

1. **Modify `internal/builder.go`**
   - Change `Build()` to return `(*BuildResult, error)` instead of `(string, error)`

2. **Modify `internal/pusher.go`**
   - Change `Push()` to return `(*PushResult, error)` instead of `error`

3. **Modify `internal/tagger.go`**
   - Change `Tag()` to return `(*TagResult, error)`
   - Change `Promote()` / `PromoteFromTag()` to return `(*PromoteResult, error)`

4. **Modify `internal/puller.go`**
   - Change `Pull()` / `PullFromTag()` to return `(*PullResult, error)`

5. **Modify `internal/current.go`**
   - `GetCurrentImage()` already returns `(string, error)` - extend to `(*CurrentResult, error)`

6. **Modify `internal/list.go`**
   - Already returns proper structs, no changes needed internally

### Phase 3: Update Command Handlers in `main.go`

Each `handle*Command()` function needs to:
1. Check the output format from global flags
2. Call the internal function (which now returns result struct)
3. On success: output result in appropriate format
4. On error: output error in appropriate format (JSON errors too!)

Example transformation:
```go
// BEFORE
func handleCurrentCommand(globalFlags *GlobalFlags, args []string) {
    // ... setup ...
    imageRef, err := currentService.GetCurrentImage(ctx, appName, environment)
    if err != nil {
        internal.LogError("Failed to get current image: %v", err)
        os.Exit(1)
    }
    fmt.Println(imageRef)
}

// AFTER  
func handleCurrentCommand(globalFlags *GlobalFlags, args []string) {
    // ... setup ...
    result, err := currentService.GetCurrentImage(ctx, appName, environment)
    if err != nil {
        outputError(globalFlags, "current", err)
        os.Exit(1)
    }
    outputResult(globalFlags, result)
}
```

### Phase 4: Handle Errors in JSON Format

When `--json` is enabled, errors should also be output as JSON:
```go
func outputError(globalFlags *GlobalFlags, command string, err error) {
    if globalFlags.JSON {
        result := CommandResult{
            Success: false,
            Command: command,
            Error:   err.Error(),
        }
        json.NewEncoder(os.Stdout).Encode(result)
    } else {
        internal.LogError("%v", err)
    }
}
```

### Phase 5: Suppress Logs in JSON Mode

When `--json` is enabled:
- Set log level to Error-only or completely suppress logs
- Progress bars should be disabled (modify spinner/progressbar instantiation)
- Only structured JSON should go to stdout

```go
// In main.go after parsing global flags
if globalFlags.JSON {
    internal.SetLogLevel(internal.LogLevelError) // Or create LogLevelNone
    // Consider: redirect stderr to /dev/null or a flag to suppress completely
}
```

---

## File Changes Summary

| File | Changes |
|------|---------|
| `main.go` | Add `--json` flag, modify all `handle*` functions |
| `internal/output.go` | **NEW** - Output format handling, result emission |
| `internal/results.go` | **NEW** - Result type definitions |
| `internal/builder.go` | Change return type to include result struct |
| `internal/pusher.go` | Change return type to include result struct |
| `internal/tagger.go` | Change return types for Tag/Promote |
| `internal/puller.go` | Change return type to include result struct |
| `internal/current.go` | Change return type to include result struct |
| `internal/list.go` | Already good - structs have JSON tags |
| `internal/log.go` | Add `LogLevelNone` or quiet mode for JSON output |

---

## Example JSON Outputs

### Success: `s3dock --json build myapp`
```json
{
  "success": true,
  "command": "build",
  "data": {
    "image_tag": "myapp:20250721-2118-f7a5a27",
    "app_name": "myapp",
    "git_hash": "f7a5a27",
    "git_time": "20250721-2118"
  }
}
```

### Success: `s3dock --json list images myapp`
```json
{
  "success": true,
  "command": "list images",
  "data": {
    "app_name": "myapp",
    "images": [
      {
        "app_name": "myapp",
        "tag": "20250721-2118-f7a5a27",
        "s3_path": "images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz",
        "year_month": "202507"
      }
    ]
  }
}
```

### Error: `s3dock --json current myapp nonexistent`
```json
{
  "success": false,
  "command": "current",
  "error": "environment pointer not found: myapp/nonexistent"
}
```

---

## Testing Strategy

1. Add unit tests for `internal/output.go` and `internal/results.go`
2. Add integration tests that verify JSON output can be parsed
3. Test error cases produce valid JSON
4. Test that `--json` suppresses progress bars and logs appropriately

---

## Alternative Considerations

1. **`--output=json` vs `--json`**: Could use `--output` with values `text`, `json`, or even `yaml`. More extensible but `--json` is simpler and matches tools like `kubectl`.

2. **Wrapper vs direct modification**: Could create wrapper types that have both Text and JSON representations. More complex but keeps internal code cleaner.

3. **Streaming JSON**: For `list` commands with many items, could use JSON Lines format. Not recommended initially - standard JSON arrays are more compatible.

---

## Implementation Order

1. **Phase 1**: Core infrastructure (output.go, results.go, global flag)
2. **Phase 3 partial**: Update `list` commands first (easiest - already have proper structs)
3. **Phase 3 partial**: Update `current` and `version` (simple, single values)
4. **Phase 2 + 3**: Update `build`, `push`, `tag`, `promote`, `pull` (require internal changes)
5. **Phase 4**: Error handling
6. **Phase 5**: Log/progress bar suppression

This phased approach allows incremental testing and keeps the scope manageable.
