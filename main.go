// markdownfmt formats Markdown.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/scanner"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kunde21/markdownfmt/v2/markdown"
	"github.com/Kunde21/markdownfmt/v2/markdownfmt"
	"github.com/pkg/diff"
)

var (
	// Main operation modes.
	list              = flag.Bool("l", false, "list files whose formatting differs from markdownfmt's")
	write             = flag.Bool("w", false, "write result to (source) file instead of stdout")
	doDiff            = flag.Bool("d", false, "display diffs instead of rewriting files")
	underlineHeadings = flag.Bool("u", false, "write underline headings instead of hashes for levels 1 and 2")
	softWraps         = flag.Bool("soft-wraps", false, "wrap lines even on soft line breaks")

	exitCode = 0
)

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	exitCode = 2
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: markdownfmt [flags] [path ...]\n")
	flag.PrintDefaults()
}

func isMarkdownFile(f os.FileInfo) bool {
	// Ignore non-Markdown files.
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && (strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown"))
}

func processFile(filename string, in io.Reader, out io.Writer) error {
	if in == nil {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	src, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	var opts []markdown.Option
	if *underlineHeadings {
		opts = append(opts, markdown.WithUnderlineHeadings())
	}
	if *softWraps {
		opts = append(opts, markdown.WithSoftWraps())
	}
	res, err := markdownfmt.Process(filename, src, opts...)
	if err != nil {
		return err
	}

	if !bytes.Equal(src, res) {
		// formatting has changed
		if *list {
			fmt.Fprintln(out, filename)
		}
		if *write {
			err = os.WriteFile(filename, res, 0)
			if err != nil {
				return err
			}
		}
		if *doDiff {
			fmt.Fprintf(os.Stderr, "diff %s markdownfmt/%s\n", filename, filename)
			err = diff.Text(
				filepath.Join("a", filename),
				filepath.Join("b", filename),
				src, res, out,
			)
			if err != nil {
				return fmt.Errorf("writing out: %s", err)
			}
		}
	}

	if !*list && !*write && !*doDiff {
		_, err = out.Write(res)
	}

	return err
}

func visitFile(path string, f os.FileInfo, err error) error {
	if err == nil && isMarkdownFile(f) {
		err = processFile(path, nil, os.Stdout)
	}
	if err != nil {
		report(err)
	}
	return nil
}

func walkDir(path string) error {
	return filepath.Walk(path, visitFile)
}

func main() {
	// call markdownfmtMain in a separate function
	// so that it can use defer and have them
	// run before the exit.
	markdownfmtMain()
	os.Exit(exitCode)
}

func markdownfmtMain() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		if err := processFile("<standard input>", os.Stdin, os.Stdout); err != nil {
			report(err)
		}
		return
	}

	for i := 0; i < flag.NArg(); i++ {
		path := flag.Arg(i)
		switch dir, err := os.Stat(path); {
		case err != nil:
			report(err)
		case dir.IsDir():
			if err := walkDir(path); err != nil {
				report(err)
			}
		default:
			if err := processFile(path, nil, os.Stdout); err != nil {
				report(err)
			}
		}
	}
}
