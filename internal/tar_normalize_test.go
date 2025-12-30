package internal

import (
	"archive/tar"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestNormalizeTar(t *testing.T) {
	fixedTime := time.Date(2025, 12, 30, 17, 18, 0, 0, time.UTC)

	tests := []struct {
		name string
		// Create a tar with different timestamps
		createTar func() *bytes.Buffer
		// Verify the normalized output
		verify func(t *testing.T, output *bytes.Buffer)
	}{
		{
			name: "normalizes file timestamps",
			createTar: func() *bytes.Buffer {
				buf := &bytes.Buffer{}
				tw := tar.NewWriter(buf)

				// Add file with different timestamp
				header := &tar.Header{
					Name:    "test.txt",
					Size:    11,
					Mode:    0644,
					ModTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				}
				tw.WriteHeader(header)
				tw.Write([]byte("hello world"))
				tw.Close()

				return buf
			},
			verify: func(t *testing.T, output *bytes.Buffer) {
				tr := tar.NewReader(output)
				header, err := tr.Next()
				if err != nil {
					t.Fatalf("Failed to read header: %v", err)
				}

				if !header.ModTime.Equal(fixedTime) {
					t.Errorf("ModTime not normalized: got %v, want %v", header.ModTime, fixedTime)
				}

				content, err := io.ReadAll(tr)
				if err != nil {
					t.Fatalf("Failed to read content: %v", err)
				}

				if string(content) != "hello world" {
					t.Errorf("Content mismatch: got %q, want %q", content, "hello world")
				}
			},
		},
		{
			name: "normalizes directory timestamps",
			createTar: func() *bytes.Buffer {
				buf := &bytes.Buffer{}
				tw := tar.NewWriter(buf)

				header := &tar.Header{
					Name:     "testdir/",
					Typeflag: tar.TypeDir,
					Mode:     0755,
					ModTime:  time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
				}
				tw.WriteHeader(header)
				tw.Close()

				return buf
			},
			verify: func(t *testing.T, output *bytes.Buffer) {
				tr := tar.NewReader(output)
				header, err := tr.Next()
				if err != nil {
					t.Fatalf("Failed to read header: %v", err)
				}

				if !header.ModTime.Equal(fixedTime) {
					t.Errorf("Directory ModTime not normalized: got %v, want %v", header.ModTime, fixedTime)
				}

				if header.Typeflag != tar.TypeDir {
					t.Errorf("Typeflag changed: got %v, want %v", header.Typeflag, tar.TypeDir)
				}
			},
		},
		{
			name: "handles multiple files",
			createTar: func() *bytes.Buffer {
				buf := &bytes.Buffer{}
				tw := tar.NewWriter(buf)

				files := []struct {
					name    string
					content string
					modTime time.Time
				}{
					{"file1.txt", "content1", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
					{"file2.txt", "content2", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)},
					{"file3.txt", "content3", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)},
				}

				for _, f := range files {
					header := &tar.Header{
						Name:    f.name,
						Size:    int64(len(f.content)),
						Mode:    0644,
						ModTime: f.modTime,
					}
					tw.WriteHeader(header)
					tw.Write([]byte(f.content))
				}
				tw.Close()

				return buf
			},
			verify: func(t *testing.T, output *bytes.Buffer) {
				tr := tar.NewReader(output)

				for i := 0; i < 3; i++ {
					header, err := tr.Next()
					if err != nil {
						t.Fatalf("Failed to read header %d: %v", i, err)
					}

					if !header.ModTime.Equal(fixedTime) {
						t.Errorf("File %d ModTime not normalized: got %v, want %v", i, header.ModTime, fixedTime)
					}
				}

				// Should be EOF
				_, err := tr.Next()
				if err != io.EOF {
					t.Errorf("Expected EOF, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.createTar()
			output := &bytes.Buffer{}

			err := NormalizeTar(input, output, fixedTime)
			if err != nil {
				t.Fatalf("NormalizeTar failed: %v", err)
			}

			tt.verify(t, output)
		})
	}
}

func TestNormalizeTar_Deterministic(t *testing.T) {
	// Create a tar with varying timestamps
	createTar := func() *bytes.Buffer {
		buf := &bytes.Buffer{}
		tw := tar.NewWriter(buf)

		// Use current time - will be different each call
		header := &tar.Header{
			Name:    "test.txt",
			Size:    5,
			Mode:    0644,
			ModTime: time.Now(),
		}
		tw.WriteHeader(header)
		tw.Write([]byte("hello"))
		tw.Close()

		return buf
	}

	fixedTime := time.Date(2025, 12, 30, 17, 18, 0, 0, time.UTC)

	// Normalize the same input multiple times
	outputs := make([][]byte, 3)
	for i := 0; i < 3; i++ {
		input := createTar()
		output := &bytes.Buffer{}

		err := NormalizeTar(input, output, fixedTime)
		if err != nil {
			t.Fatalf("NormalizeTar failed on run %d: %v", i, err)
		}

		outputs[i] = output.Bytes()
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		if !bytes.Equal(outputs[0], outputs[i]) {
			t.Errorf("Output %d differs from output 0", i)
		}
	}
}

func TestNormalizeTar_EmptyTar(t *testing.T) {
	// Create empty tar
	input := &bytes.Buffer{}
	tw := tar.NewWriter(input)
	tw.Close()

	output := &bytes.Buffer{}
	fixedTime := time.Date(2025, 12, 30, 17, 18, 0, 0, time.UTC)

	err := NormalizeTar(input, output, fixedTime)
	if err != nil {
		t.Fatalf("NormalizeTar failed on empty tar: %v", err)
	}

	// Should produce valid empty tar
	tr := tar.NewReader(output)
	_, err = tr.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF for empty tar, got: %v", err)
	}
}

func TestNormalizeTar_InvalidTar(t *testing.T) {
	// Create invalid tar data
	input := bytes.NewBufferString("this is not a valid tar file")
	output := &bytes.Buffer{}
	fixedTime := time.Date(2025, 12, 30, 17, 18, 0, 0, time.UTC)

	err := NormalizeTar(input, output, fixedTime)
	if err == nil {
		t.Error("Expected error for invalid tar, got nil")
	}

	if !strings.Contains(err.Error(), "error reading tar header") {
		t.Errorf("Expected tar reading error, got: %v", err)
	}
}

func TestParseGitTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTime  time.Time
		wantError bool
	}{
		{
			name:     "valid git time",
			input:    "20251230-1718",
			wantTime: time.Date(2025, 12, 30, 17, 18, 0, 0, time.UTC),
		},
		{
			name:     "valid git time - midnight",
			input:    "20250101-0000",
			wantTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "valid git time - end of day",
			input:    "20251231-2359",
			wantTime: time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC),
		},
		{
			name:      "invalid format - missing dash",
			input:     "202512301718",
			wantError: true,
		},
		{
			name:      "invalid format - wrong length",
			input:     "2025-12-30",
			wantError: true,
		},
		{
			name:      "invalid format - letters",
			input:     "abcd1230-1718",
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, err := ParseGitTime(tt.input)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !gotTime.Equal(tt.wantTime) {
				t.Errorf("ParseGitTime() = %v, want %v", gotTime, tt.wantTime)
			}
		})
	}
}
