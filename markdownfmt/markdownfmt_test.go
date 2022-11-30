package markdownfmt_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kunde21/markdownfmt/v2/markdown"
	"github.com/Kunde21/markdownfmt/v2/markdownfmt"
	"github.com/google/go-cmp/cmp"
)

func TestSame(t *testing.T) {
	matches, err := filepath.Glob("testfiles/*.same.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			reference, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", reference)
			if err != nil {
				t.Fatal(err)
			}

			diff := diff(reference, output)
			if diff != "" {
				t.Errorf("Difference in %s of %d lines:\n%s", f, strings.Count(diff, "\n"), diff)
			}
		})
	}
}

func TestWithHardWraps(t *testing.T) {
	matches, err := filepath.Glob("testfiles/*same-softwrap.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			reference, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", reference, markdown.WithSoftWraps())
			if err != nil {
				t.Fatal(err)
			}

			diff := diff(reference, output)
			if len(diff) != 0 {
				t.Errorf("Difference in %s of %d lines:\n%s", f, strings.Count(diff, "\n"), diff)
			}
		})
	}
}

func TestSameUnderline(t *testing.T) {
	matches, err := filepath.Glob("testfiles/*.same-underline.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			reference, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", reference, markdown.WithUnderlineHeadings())
			if err != nil {
				t.Fatal(err)
			}

			diff := diff(reference, output)
			if len(diff) != 0 {
				t.Errorf("Difference in %s of %d lines:\n%s", f, strings.Count(diff, "\n"), diff)
			}
		})
	}
}

func TestDifferent(t *testing.T) {
	matches, err := filepath.Glob("testfiles/*.input.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			expOutput, err := os.ReadFile(strings.ReplaceAll(f, ".input.md", ".output.md"))
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", input)
			if err != nil {
				t.Fatal(err)
			}

			diff := diff(expOutput, output)
			if len(diff) != 0 {
				fmt.Println("----\n", string(output), "\n---")

				t.Errorf("Difference in %s of %d lines:\n%s", f, strings.Count(diff, "\n"), diff)
			}
		})
	}
}

func TestCustomCodeFormatter(t *testing.T) {
	reference, err := os.ReadFile("testfiles/nested-code.same.md")
	if err != nil {
		t.Fatal(err)
	}

	output, err := markdownfmt.Process(
		"", reference, markdown.WithCodeFormatters(markdown.CodeFormatter{
			Name: "Makefile",
			Format: func(b []byte) []byte {
				return []byte("replaced contents")
			},
		}))
	if err != nil {
		t.Fatal(err)
	}

	if want := " replaced contents\n"; !bytes.Contains(output, []byte(want)) {
		t.Errorf("output does not contain %q:\n%s", want, output)
	}
}

func diff(want, got []byte) string {
	return cmp.Diff(string(want), string(got))
}
