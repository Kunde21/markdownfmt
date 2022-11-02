package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestEmphasisAndStrongToken(t *testing.T) {
	type testCase struct{ give, want string }

	tests := []struct {
		desc   string
		emph   rune   // emphasis token; use 0 for default
		strong string // strong token; use "" for default
		cases  []testCase
	}{
		{
			// By default, we use '*' for everything.
			desc: "default",
			cases: []testCase{
				{"_bar_", "*bar*"},
				{"__bar__", "**bar**"},
				{"___bar___", "***bar***"},
				{"*__bar__*", "***bar***"},
				{"__*bar*__", "***bar***"},
			},
		},
		{
			// '*' may be specified explicitly.
			desc: "emph_star",
			emph: '*',
			cases: []testCase{
				{"_bar_", "*bar*"},
				{"__bar__", "**bar**"},
				{"___bar___", "***bar***"},
				{"*__bar__*", "***bar***"},
				{"__*bar*__", "***bar***"},
			},
		},
		{
			// If WithEmphasisToken('_') is specified,
			// all versions use underscore for everything.
			desc: "emph_underscore",
			emph: '_',
			cases: []testCase{
				{"*bar*", "_bar_"},
				{"**bar**", "__bar__"},
				{"***bar***", "___bar___"},
				{"_**bar**_", "___bar___"},
				{"**_bar_**", "___bar___"},
			},
		},
		{
			// WithStrongToken("__") may be specified without
			// changing emph.
			desc:   "strong_underscore",
			strong: "__",
			cases: []testCase{
				{"*bar*", "*bar*"},
				{"**bar**", "__bar__"},
				{"***bar***", "*__bar__*"},
				{"_**bar**_", "*__bar__*"},
				{"**_bar_**", "__*bar*__"},
			},
		},
		{
			// WithEmphasisToken('_'), WithStrongToken("**")
			desc:   "emph_underscore/strong_stars",
			emph:   '_',
			strong: "**",
			cases: []testCase{
				{"*bar*", "_bar_"},
				{"__bar__", "**bar**"},
				// goldmark sees "___[...]___" as
				// <emph<strong>[...]</strong></emph>.
				{"___bar___", "_**bar**_"},
				{"*__bar__*", "_**bar**_"},
				{"__*bar*__", "**_bar_**"},
			},
		},
	}

	runTestCase := func(renderer *Renderer, tc testCase) {
		src := []byte(tc.give)
		node := goldmark.DefaultParser().Parse(text.NewReader(src))
		var buff bytes.Buffer
		if err := renderer.Render(&buff, src, node); err != nil {
			t.Fatal(err)
		}

		// We omit the trailing newline from test cases for
		// convenience.
		want := strings.TrimSuffix(tc.want, "\n")
		got := strings.TrimSuffix(buff.String(), "\n")

		if diff := cmp.Diff(want, got); len(diff) > 0 {
			t.Error(diff)
		}
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			renderer := NewRenderer()
			if tt.emph != 0 {
				renderer.AddMarkdownOptions(WithEmphasisToken(tt.emph))
			}
			if tt.strong != "" {
				renderer.AddMarkdownOptions(WithStrongToken(tt.strong))
			}

			for _, tc := range tt.cases {
				t.Run(tc.give, func(t *testing.T) {
					runTestCase(renderer, tc)
				})
			}
		})
	}
}
