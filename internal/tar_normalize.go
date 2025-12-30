package internal

import (
	"archive/tar"
	"fmt"
	"io"
	"time"
)

// NormalizeTar reads a tar stream and rewrites all timestamps to fixedTime.
// This makes Docker image exports deterministic by removing timestamp variations
// that occur on each 'docker save' execution.
//
// The function:
// - Reads headers from the input tar stream
// - Sets ModTime to fixedTime for all entries
// - Clears AccessTime and ChangeTime (not supported in USTAR format)
// - Writes normalized headers and content to output
//
// The resulting tar is fully compatible with Docker and can be loaded with 'docker load'.
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
			return fmt.Errorf("error reading tar header: %w", err)
		}

		// Normalize ModTime to the fixed time
		// Note: AccessTime and ChangeTime are not supported in USTAR format
		// (which Docker uses), so we clear them to avoid encoding errors
		header.ModTime = fixedTime
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}

		// Write normalized header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("error writing tar header: %w", err)
		}

		// Copy file content if it's a regular file
		if header.Typeflag == tar.TypeReg {
			if _, err := io.Copy(tarWriter, tarReader); err != nil {
				return fmt.Errorf("error copying file content: %w", err)
			}
		}
	}

	return nil
}

// ParseGitTime converts a git timestamp string (format: YYYYMMDD-HHMM) to time.Time
func ParseGitTime(gitTime string) (time.Time, error) {
	// Format: 20251230-1718
	const layout = "20060102-1504"
	t, err := time.Parse(layout, gitTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid git time format %q: %w", gitTime, err)
	}
	return t, nil
}
