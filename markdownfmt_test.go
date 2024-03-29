package markdownfmt_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Kunde21/markdownfmt/v3"
	"github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark/text"
)

func TestSame(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.same.md")
	require.NoError(t, err)

	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			reference, err := os.ReadFile(f)
			require.NoError(t, err)

			output, err := markdownfmt.Process("", reference)
			require.NoError(t, err)

			assert.Equal(t, string(reference), string(output))
		})
	}
}

func TestWithHardWraps(t *testing.T) {
	matches, err := filepath.Glob("testdata/*same-softwrap.md")
	require.NoError(t, err)

	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			reference, err := os.ReadFile(f)
			require.NoError(t, err)

			output, err := markdownfmt.Process("", reference, markdown.WithSoftWraps())
			require.NoError(t, err)

			assert.Equal(t, string(reference), string(output))
		})
	}
}

func TestSameUnderline(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.same-underline.md")
	require.NoError(t, err)

	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			reference, err := os.ReadFile(f)
			require.NoError(t, err)

			output, err := markdownfmt.Process("", reference, markdown.WithUnderlineHeadings())
			require.NoError(t, err)

			assert.Equal(t, string(reference), string(output))
		})
	}
}

func TestDifferent(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.input.md")
	require.NoError(t, err)

	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			expOutput, err := os.ReadFile(strings.ReplaceAll(f, ".input.md", ".output.md"))
			require.NoError(t, err)

			output, err := markdownfmt.Process("", input)
			require.NoError(t, err)

			assert.Equal(t, string(expOutput), string(output))
		})
	}
}

func TestGoCodeFormatter(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.gofmt-input.md")
	require.NoError(t, err)

	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			expOutput, err := os.ReadFile(strings.ReplaceAll(f, ".gofmt-input.md", ".gofmt-output.md"))
			require.NoError(t, err)

			output, err := markdownfmt.Process("", input, markdown.WithCodeFormatters(markdown.GoCodeFormatter))
			require.NoError(t, err)

			assert.Equal(t, string(expOutput), string(output))
		})
	}
}

func TestListIndentUniform(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.list-uniform-input.md")
	require.NoError(t, err)

	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			expOutput, err := os.ReadFile(strings.ReplaceAll(f, ".list-uniform-input.md", ".list-uniform-output.md"))
			require.NoError(t, err)

			output, err := markdownfmt.Process("", input, markdown.WithListIndentStyle(markdown.ListIndentUniform))
			require.NoError(t, err)

			assert.Equal(t, string(expOutput), string(output))
		})
	}
}

func TestListIndentUniform_ReparseOutput(t *testing.T) {
	// This test verifies that there is no change in semantics
	// between the "aligned" and "uniform" indentation styles.
	//
	// To do this,
	// it processes the uniform inputs with aligned indentation
	// and compares the output of that with the output of
	// re-processing the uniform outputs with aligned indentation.
	//
	// That is, it verifies:
	//
	//	aligned(txt) == aligned(uniform(txt))

	matches, err := filepath.Glob("testdata/*.list-uniform-input.md")
	require.NoError(t, err)

	for _, inputFile := range matches {
		t.Run(inputFile, func(t *testing.T) {
			input, err := os.ReadFile(inputFile)
			require.NoError(t, err)
			alignedOutput, err := markdownfmt.Process("", input, markdown.WithListIndentStyle(markdown.ListIndentAligned))
			require.NoError(t, err)

			outputFile := strings.ReplaceAll(inputFile, ".list-uniform-input.md", ".list-uniform-output.md")
			uniformOutput, err := os.ReadFile(outputFile)
			require.NoError(t, err)
			realignedOutput, err := markdownfmt.Process("", uniformOutput, markdown.WithListIndentStyle(markdown.ListIndentAligned))
			require.NoError(t, err)

			assert.Equal(t, string(alignedOutput), string(realignedOutput))
		})
	}
}

func TestCustomCodeFormatter(t *testing.T) {
	reference, err := os.ReadFile("testdata/nested-code.same.md")
	require.NoError(t, err)

	output, err := markdownfmt.Process(
		"", reference, markdown.WithCodeFormatters(markdown.CodeFormatter{
			Name: "Makefile",
			Format: func(b []byte) []byte {
				return []byte("replaced contents")
			},
		}))
	require.NoError(t, err)

	assert.Contains(t, string(output), " replaced contents\n")
}

func BenchmarkRender(b *testing.B) {
	inputs, err := filepath.Glob("testdata/*.input.md")
	require.NoError(b, err)

	sames, err := filepath.Glob("testdata/*.same.md")
	require.NoError(b, err)

	matches := append(inputs, sames...)
	sort.Strings(matches)

	for _, fname := range matches {
		b.Run(filepath.Base(fname), func(b *testing.B) {
			src, err := os.ReadFile(fname)
			require.NoError(b, err)

			md := markdownfmt.NewGoldmark(
				// Disable code formatters.
				// We're not benchmarking gofmt.
				markdown.WithCodeFormatters(),
			)
			doc := md.Parser().Parse(text.NewReader(src))
			renderer := md.Renderer()

			var buff bytes.Buffer

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buff.Reset()

				err := renderer.Render(&buff, src, doc)
				require.NoError(b, err)
			}
		})
	}
}
