package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestListIndentStyle_Aligned(t *testing.T) {
	renderer := NewRenderer()
	renderer.AddOptions(WithListIndentStyle(ListIndentAligned))

	tests := []struct {
		desc string
		give string
		want string
	}{
		{
			desc: "no nest",
			give: "- foo\n",
			want: "- foo\n",
		},
		{
			desc: "multiple paragraphs",
			give: joinLines(
				"- foo",
				"",
				"    bar",
			),
			want: joinLines(
				"- foo",
				"",
				"  bar",
			),
		},
		{
			desc: "nested code",
			give: joinLines(
				"- foo",
				"",
				"    ```go",
				"    func main()",
				"    ```",
			),
			want: joinLines(
				"- foo",
				"",
				"  ```go",
				"  func main()",
				"  ```",
			),
		},
		{
			desc: "nested list",
			give: joinLines(
				"- foo",
				"",
				"    - bar",
				"    - baz",
				"",
				"- qux",
			),
			want: joinLines(
				"- foo",
				"",
				"  - bar",
				"  - baz",
				"",
				"- qux",
			),
		},
		{
			desc: "long number",
			give: joinLines(
				"123. foo",
				"",
				"      bar",
			),
			want: joinLines(
				"123. foo",
				"",
				"     bar",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			src := []byte(tt.give)
			node := goldmark.DefaultParser().Parse(text.NewReader(src))

			var buff bytes.Buffer
			require.NoError(t, renderer.Render(&buff, src, node))
			got := buff.String()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListIndentStyle_Uniform(t *testing.T) {
	renderer := NewRenderer()
	renderer.AddOptions(WithListIndentStyle(ListIndentUniform))

	tests := []struct {
		desc string
		give string
		want string
	}{
		{
			desc: "no nest",
			give: "- foo\n",
			want: "- foo\n",
		},
		{
			desc: "multiple paragraphs",
			give: joinLines(
				"- foo",
				"",
				"  bar",
			),
			want: joinLines(
				"- foo",
				"",
				"    bar",
			),
		},
		{
			desc: "nested code",
			give: joinLines(
				"- foo",
				"",
				"  ```go",
				"  func main()",
				"  ```",
			),
			want: joinLines(
				"- foo",
				"",
				"    ```go",
				"    func main()",
				"    ```",
			),
		},
		{
			desc: "nested list",
			give: joinLines(
				"- foo",
				"",
				"  - bar",
				"  - baz",
				"",
				"- qux",
			),
			want: joinLines(
				"- foo",
				"",
				"    - bar",
				"    - baz",
				"",
				"- qux",
			),
		},
		{
			desc: "long number",
			give: joinLines(
				"123. foo",
				"",
				"     bar",
			),
			want: joinLines(
				"123. foo",
				"",
				"     bar",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			src := []byte(tt.give)
			node := goldmark.DefaultParser().Parse(text.NewReader(src))

			var buff bytes.Buffer
			require.NoError(t, renderer.Render(&buff, src, node))
			got := buff.String()

			assert.Equal(t, tt.want, got)
		})
	}
}

// Joins one or more lines, ending with a trailing newline if absent.
func joinLines(lines ...string) string {
	s := strings.Join(lines, "\n")
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}
