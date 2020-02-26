package markdown_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kunde21/markdownfmt/markdown"
)

func TestSame(t *testing.T) {
	matches, err := filepath.Glob("testfiles/*.same.md")
	if err != nil {
		log.Fatalln(err)
	}
	for _, f := range matches {
		reference, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalln(err)
		}

		output, err := markdown.Process("", []byte(reference))
		if err != nil {
			log.Fatalln(err)
		}

		diff, err := diff([]byte(reference), output)
		if err != nil {
			log.Fatalln(err)
		}

		if len(diff) != 0 {
			t.Errorf("Difference in %s of %d lines:\n%s", f, bytes.Count(diff, []byte("\n")), string(diff))
		}
	}
}

func TestDifferent(t *testing.T) {
	matches, err := filepath.Glob("testfiles/*.input.md")
	if err != nil {
		log.Fatalln(err)
	}
	for _, f := range matches {
		input, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalln(err)
		}

		outputname := strings.ReplaceAll(f, "input.md", "output.md")
		expOutput, err := ioutil.ReadFile(outputname)
		if err != nil {
			log.Fatalln(err)
		}

		output, err := markdown.Process("", input)
		if err != nil {
			log.Fatalln(err)
		}

		diff, err := diff(expOutput, output)
		if err != nil {
			log.Fatalln(err)
		}

		if len(diff) != 0 {
			t.Errorf("Difference in %s of %d lines:\n%s", f, bytes.Count(diff, []byte("\n")), string(diff))
		}
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

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}
