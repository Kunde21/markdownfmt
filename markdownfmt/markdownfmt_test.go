package markdownfmt_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kunde21/markdownfmt/v2/markdown"
	"github.com/Kunde21/markdownfmt/v2/markdownfmt"
)

func TestSame(t *testing.T) {
	matches, err := filepath.Glob("testfiles/*.same.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range matches {
		t.Run(f, func(t *testing.T) {
			reference, err := ioutil.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", reference)
			if err != nil {
				t.Fatal(err)
			}

			diff, err := diff(reference, output)
			if err != nil {
				t.Fatal(err)
			}

			if len(diff) != 0 {
				t.Errorf("Difference in %s of %d lines:\n%s", f, bytes.Count(diff, []byte("\n")), string(diff))
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
			reference, err := ioutil.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", reference, markdown.WithSoftWraps())
			if err != nil {
				t.Fatal(err)
			}

			diff, err := diff(reference, output)
			if err != nil {
				t.Fatal(err)
			}

			if len(diff) != 0 {
				t.Errorf("Difference in %s of %d lines:\n%s", f, bytes.Count(diff, []byte("\n")), string(diff))
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
			reference, err := ioutil.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", reference, markdown.WithUnderlineHeadings())
			if err != nil {
				t.Fatal(err)
			}

			diff, err := diff(reference, output)
			if err != nil {
				t.Fatal(err)
			}

			if len(diff) != 0 {
				t.Errorf("Difference in %s of %d lines:\n%s", f, bytes.Count(diff, []byte("\n")), string(diff))
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
			input, err := ioutil.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			expOutput, err := ioutil.ReadFile(strings.ReplaceAll(f, ".input.md", ".output.md"))
			if err != nil {
				t.Fatal(err)
			}

			output, err := markdownfmt.Process("", input)
			if err != nil {
				t.Fatal(err)
			}

			diff, err := diff(expOutput, output)
			if err != nil {
				t.Fatal(err)
			}

			if len(diff) != 0 {
				fmt.Println("----\n", string(output), "\n---")

				t.Errorf("Difference in %s of %d lines:\n%s", f, bytes.Count(diff, []byte("\n")), string(diff))
			}
		})
	}
}

func TestCustomCodeFormatter(t *testing.T) {
	reference, err := ioutil.ReadFile("testfiles/nested-code.same.md")
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

// TODO: Factor out.
func diff(b1, b2 []byte) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "markdownfmt")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "markdownfmt")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	_, err = f1.Write(b1)
	if err != nil {
		return
	}
	_, err = f2.Write(b2)
	if err != nil {
		return
	}

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}
