package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStdinStdout(t *testing.T) {
	tests := []struct {
		desc       string
		args       []string
		stdin      string
		wantStdout string
	}{
		{
			desc:       "simple",
			stdin:      "# hello\nworld",
			wantStdout: "# hello\n\nworld\n",
		},
		{
			desc:       "go code/no gofmt",
			stdin:      "```go\nfunc main(){fmt.Println(42)\n}\n```",
			wantStdout: "```go\nfunc main(){fmt.Println(42)\n}\n```\n",
		},
		{
			// The code formatters feature is tested fully elsewhere.
			// This is just to verify that the '-gofmt' flag
			// has the desired effect.
			desc:       "go code/gofmt",
			args:       []string{"-gofmt"},
			stdin:      "```go\nfunc main(){fmt.Println(42)\n}\n```",
			wantStdout: "```go\nfunc main() {\n\tfmt.Println(42)\n}\n```\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := mainCmd{
				Stdin:  strings.NewReader(tt.stdin),
				Stdout: &stdout,
				Stderr: &stderr,
			}
			cmd.Run(tt.args)
			if want, got := 0, cmd.exitCode; want != got {
				t.Fatalf("unexpected exit code %v, want %v", got, want)
			}
			if stderr.Len() > 0 {
				t.Errorf("unexpected stderr: %v", stderr.String())
			}

			gotOutput := stdout.String()
			if diff := cmp.Diff(tt.wantStdout, gotOutput); len(diff) > 0 {
				t.Errorf("unexpected output %q, want %q\ndiff: %v", gotOutput, tt.wantStdout, diff)
			}
		})
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

func TestHelp(t *testing.T) {
	var stderr bytes.Buffer
	cmd := mainCmd{
		Stdin:  new(bytes.Buffer), // empty stdin
		Stdout: io.Discard,
		Stderr: &stderr,
	}
	cmd.Run([]string{"-h"})

	// Exit code for --help must be zero.
	if want, got := 0, cmd.exitCode; want != got {
		t.Errorf("unexpected exit code %v, want %v", got, want)
	}

	got := stderr.String()
	want := "markdownfmt [flags] [path"
	if !strings.Contains(got, want) {
		t.Errorf("stderr does not contain %q:\n%v", want, got)
	}
}

func TestParseArgs(t *testing.T) {
	type flags struct {
		list              bool
		write             bool
		diff              bool
		underlineHeadings bool
		softWraps         bool
		gofmt             bool
	}

	tests := []struct {
		desc string
		give []string

		want     flags
		wantArgs []string
	}{
		{
			desc: "no arguments",
			give: []string{},
		},
		{
			desc: "list",
			give: []string{"-l"},
			want: flags{list: true},
		},
		{
			desc: "write",
			give: []string{"-w"},
			want: flags{write: true},
		},
		{
			desc: "diff",
			give: []string{"-d"},
			want: flags{diff: true},
		},
		{
			desc: "underlineHeadings",
			give: []string{"-u"},
			want: flags{underlineHeadings: true},
		},
		{
			desc: "softWraps",
			give: []string{"-soft-wraps"},
			want: flags{softWraps: true},
		},
		{
			desc: "gofmt",
			give: []string{"-gofmt"},
			want: flags{gofmt: true},
		},
		{
			desc:     "file name with flags",
			give:     []string{"-w", "foo.md", "bar/", "baz.md"},
			want:     flags{write: true},
			wantArgs: []string{"foo.md", "bar/", "baz.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Hack:
			// If tt.wantArgs == nil, replace it with an empty slice.
			// These are both equivalent,
			// but cmp.Diff differentiates between the two by default.
			if tt.wantArgs == nil {
				tt.wantArgs = make([]string, 0)
			}

			var stdout, stderr bytes.Buffer
			cmd := mainCmd{
				Stdin:  new(bytes.Buffer), // empty stdin
				Stdout: &stdout,
				Stderr: &stderr,
			}

			gotArgs, err := cmd.parseArgs(tt.give)
			if err != nil {
				t.Fatal(err)
			}

			if stderr.Len() > 0 {
				t.Errorf("unexpected stderr: %v", stderr.String())
			}

			if want, got := tt.want.list, cmd.list; want != got {
				t.Errorf("incorrect %v: %v, want %v", "list", want, got)
			}
			if want, got := tt.want.write, cmd.write; want != got {
				t.Errorf("incorrect %v: %v, want %v", "write", want, got)
			}
			if want, got := tt.want.diff, cmd.diff; want != got {
				t.Errorf("incorrect %v: %v, want %v", "diff", want, got)
			}
			if want, got := tt.want.underlineHeadings, cmd.underlineHeadings; want != got {
				t.Errorf("incorrect %v: %v, want %v", "underlineHeadings", want, got)
			}
			if want, got := tt.want.softWraps, cmd.softWraps; want != got {
				t.Errorf("incorrect %v: %v, want %v", "softWraps", want, got)
			}
			if want, got := tt.want.gofmt, cmd.gofmt; want != got {
				t.Errorf("incorrect %v: %v, want %v", "gofmt", want, got)
			}
			if diff := cmp.Diff(tt.wantArgs, gotArgs); len(diff) > 0 {
				t.Errorf("incorrect args: %q, want %q\ndiff: %v", gotArgs, tt.wantArgs, diff)
			}
		})
	}
}

func TestParseArgs_UnknownFlag(t *testing.T) {
	var stderr bytes.Buffer
	cmd := mainCmd{
		Stdin:  new(bytes.Buffer), // empty stdin
		Stdout: io.Discard,
		Stderr: &stderr,
	}

	_, err := cmd.parseArgs([]string{"-unknown-flag"})
	if err == nil {
		t.Fatal("Expected failure")
	}

	got := stderr.String()
	want := "flag provided but not defined: -unknown-flag"
	if !strings.Contains(got, want) {
		t.Errorf("error does not contain %q:\n%v", want, got)
	}
}
