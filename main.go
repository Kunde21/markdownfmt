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
)

func (cmd *mainCmd) report(err error) {
	scanner.PrintError(cmd.Stderr, err)
	cmd.exitCode = 2
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

func (cmd *mainCmd) processFile(filename string, in io.Reader, out io.Writer) error {
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
			fmt.Fprintf(cmd.Stderr, "diff %s markdownfmt/%s\n", filename, filename)
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

func (cmd *mainCmd) visitFile(path string, f os.FileInfo, err error) error {
	if err == nil && isMarkdownFile(f) {
		err = cmd.processFile(path, nil, cmd.Stdout)
	}
	if err != nil {
		cmd.report(err)
	}
	return nil
}

func (cmd *mainCmd) walkDir(path string) error {
	return filepath.Walk(path, cmd.visitFile)
}

func main() {
	cmd := mainCmd{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	// put core logic in a separate function
	// so that it can use defer and have them
	// run before the exit.
	cmd.Run(os.Args[1:])
	os.Exit(cmd.exitCode)
}

type mainCmd struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	exitCode int
}

func (cmd *mainCmd) Run(args []string) {
	flag.Usage = usage
	flag.CommandLine.Parse(args)

	if flag.NArg() == 0 {
		if err := cmd.processFile("<standard input>", cmd.Stdin, cmd.Stdout); err != nil {
			cmd.report(err)
		}
		return
	}

	for i := 0; i < flag.NArg(); i++ {
		path := flag.Arg(i)
		switch dir, err := os.Stat(path); {
		case err != nil:
			cmd.report(err)
		case dir.IsDir():
			if err := cmd.walkDir(path); err != nil {
				cmd.report(err)
			}
		default:
			if err := cmd.processFile(path, nil, cmd.Stdout); err != nil {
				cmd.report(err)
			}
		}
	}
}
