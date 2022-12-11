package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStdinStdout(t *testing.T) {
	const (
		input  = "# hello\nworld"
		output = "# hello\n\nworld\n"
	)

	var stdout, stderr bytes.Buffer
	cmd := mainCmd{
		Stdin:  strings.NewReader(input),
		Stdout: &stdout,
		Stderr: &stderr,
	}
	cmd.Run(nil /* args */)
	if want, got := 0, cmd.exitCode; want != got {
		t.Fatalf("unexpected exit code %v, want %v", got, want)
	}

	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %v", stderr.String())
	}

	gotOutput := stdout.String()
	if diff := cmp.Diff(output, gotOutput); len(diff) > 0 {
		t.Errorf("unexpected output %q, want %q\ndiff: %v", gotOutput, output, diff)
	}
}

func TestFileDoesNotExist(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := mainCmd{
		Stdin:  new(bytes.Buffer), // empty stdin
		Stdout: &stdout,
		Stderr: &stderr,
	}
	cmd.Run([]string{"file-does-not-exist.md"})

	if want, got := 2, cmd.exitCode; want != got {
		t.Errorf("unexpected exit code %v, want %v", got, want)
	}

	gotStderr := stderr.String()
	wantStderr := "file-does-not-exist.md: no such file"
	if !strings.Contains(gotStderr, wantStderr) {
		t.Errorf("unexpected stderr %q, should contain %q", gotStderr, wantStderr)
	}
}
