package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			desc:       "list-indent-style",
			args:       []string{"-list-indent-style", "uniform"},
			stdin:      "- foo\n  - bar\n- baz\n",
			wantStdout: "- foo\n    - bar\n- baz\n",
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
			assert.Zero(t, cmd.exitCode)
			assert.Empty(t, stderr.String())
			assert.Equal(t, tt.wantStdout, stdout.String())
		})
	}
}

func TestFileDoesNotExist(t *testing.T) {
	var stderr bytes.Buffer
	cmd := mainCmd{
		Stdin:  new(bytes.Buffer), // empty stdin
		Stdout: io.Discard,
		Stderr: &stderr,
	}
	cmd.Run([]string{"file-does-not-exist.md"})

	assert.Equal(t, 2, cmd.exitCode)
	assert.Contains(t, stderr.String(), "file-does-not-exist.md: no such file")
}

func TestHelp(t *testing.T) {
	var stderr bytes.Buffer
	cmd := mainCmd{
		Stdin:  new(bytes.Buffer), // empty stdin
		Stdout: io.Discard,
		Stderr: &stderr,
	}
	cmd.Run([]string{"-h"})

	assert.Zero(t, cmd.exitCode, "exit code for --help must be zero")
	assert.Contains(t, stderr.String(), "markdownfmt [flags] [path")
}

func TestParseArgs(t *testing.T) {
	type flags struct {
		list              bool
		write             bool
		diff              bool
		underlineHeadings bool
		softWraps         bool
		gofmt             bool
		listIndentStyle   markdown.ListIndentStyle
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
			desc: "list indent style/aligned",
			give: []string{"-list-indent-style=aligned"},
			want: flags{listIndentStyle: markdown.ListIndentAligned},
		},
		{
			desc: "list indent style/uniform",
			give: []string{"-list-indent-style=uniform"},
			want: flags{listIndentStyle: markdown.ListIndentUniform},
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
			require.NoError(t, err)
			assert.Empty(t, stderr.String(), "incorrect stderr")

			assert.Equal(t, tt.want.list, cmd.list, "list")
			assert.Equal(t, tt.want.write, cmd.write, "write")
			assert.Equal(t, tt.want.diff, cmd.diff, "diff")
			assert.Equal(t, tt.want.underlineHeadings, cmd.underlineHeadings, "underlineHeadings")
			assert.Equal(t, tt.want.softWraps, cmd.softWraps, "softWraps")
			assert.Equal(t, tt.want.gofmt, cmd.gofmt, "gofmt")
			assert.Equal(t, tt.want.listIndentStyle, cmd.listIndentStyle, "listIndentStyle")
			assert.Equal(t, tt.wantArgs, gotArgs, "args")
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
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "flag provided but not defined: -unknown-flag")
}

func TestParseArgs_UnknownListIndentStyle(t *testing.T) {
	var stderr bytes.Buffer
	cmd := mainCmd{
		Stdin:  new(bytes.Buffer), // empty stdin
		Stdout: io.Discard,
		Stderr: &stderr,
	}

	_, err := cmd.parseArgs([]string{"-list-indent-style=whatisthis"})
	require.Error(t, err)
	assert.Contains(t, stderr.String(), `invalid value "whatisthis"`)
	assert.Contains(t, stderr.String(), `unrecognized style "whatisthis"`)
}
