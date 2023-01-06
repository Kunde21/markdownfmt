// markdownfmt formats Markdown.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/scanner"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kunde21/markdownfmt/v3"
	"github.com/Kunde21/markdownfmt/v3/markdown"
	"github.com/pkg/diff"
)

type listIndentStyle markdown.ListIndentStyle

var _ flag.Getter = (*listIndentStyle)(nil)

func (s *listIndentStyle) Get() interface{} {
	return markdown.ListIndentStyle(*s)
}

func (s *listIndentStyle) String() string {
	switch markdown.ListIndentStyle(*s) {
	case markdown.ListIndentAligned:
		return "aligned"
	case markdown.ListIndentUniform:
		return "uniform"
	default:
		return "invalid"
	}
}

func (s *listIndentStyle) Set(v string) error {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "aligned":
		*s = listIndentStyle(markdown.ListIndentAligned)
	case "uniform":
		*s = listIndentStyle(markdown.ListIndentUniform)
	default:
		return fmt.Errorf(`unrecognized style %q: valid values are "aligned" and "uniform"`, v)
	}
	return nil
}

func (cmd *mainCmd) registerFlags(flag *flag.FlagSet) {
	flag.BoolVar(&cmd.list, "l", false, "list files whose formatting differs from markdownfmt's")
	flag.BoolVar(&cmd.write, "w", false, "write result to (source) file instead of stdout")
	flag.BoolVar(&cmd.diff, "d", false, "display diffs instead of rewriting files")
	flag.BoolVar(&cmd.underlineHeadings, "u", false, "write underline headings instead of hashes for levels 1 and 2")
	flag.BoolVar(&cmd.softWraps, "soft-wraps", false, "wrap lines even on soft line breaks")
	flag.BoolVar(&cmd.gofmt, "gofmt", false, "reformat Go source inside fenced code blocks")
	flag.Var((*listIndentStyle)(&cmd.listIndentStyle), "list-indent-style", `style for indenting items inside lists ("aligned" or "uniform")`)
}

func (cmd *mainCmd) report(err error) {
	scanner.PrintError(cmd.Stderr, err)
	cmd.exitCode = 2
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

	opts := []markdown.Option{markdown.WithListIndentStyle(cmd.listIndentStyle)}
	if cmd.underlineHeadings {
		opts = append(opts, markdown.WithUnderlineHeadings())
	}
	if cmd.softWraps {
		opts = append(opts, markdown.WithSoftWraps())
	}
	if cmd.gofmt {
		opts = append(opts, markdown.WithCodeFormatters(markdown.GoCodeFormatter))
	}
	res, err := markdownfmt.Process(filename, src, opts...)
	if err != nil {
		return err
	}

	if !bytes.Equal(src, res) {
		// formatting has changed
		if cmd.list {
			fmt.Fprintln(out, filename)
		}
		if cmd.write {
			err = os.WriteFile(filename, res, 0)
			if err != nil {
				return err
			}
		}
		if cmd.diff {
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

	if !cmd.list && !cmd.write && !cmd.diff {
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

	// Command line flags:

	// Main operation modes.
	list  bool
	write bool
	diff  bool

	// Output manipulation.
	underlineHeadings bool
	softWraps         bool
	gofmt             bool
	listIndentStyle   markdown.ListIndentStyle
}

func (cmd *mainCmd) parseArgs(args []string) ([]string, error) {
	flag := flag.NewFlagSet("markdownfmt", flag.ContinueOnError)
	flag.SetOutput(cmd.Stderr)
	flag.Usage = func() {
		fmt.Fprintln(cmd.Stderr, "usage: markdownfmt [flags] [path ...]")
		flag.PrintDefaults()
	}
	cmd.registerFlags(flag)
	err := flag.Parse(args)
	return flag.Args(), err
}

func (cmd *mainCmd) Run(args []string) {
	args, err := cmd.parseArgs(args)
	if err != nil {
		// --help exits with a 0 status code.
		if !errors.Is(err, flag.ErrHelp) {
			cmd.exitCode = 2
		}
		return
	}

	if len(args) == 0 {
		if err := cmd.processFile("<standard input>", cmd.Stdin, cmd.Stdout); err != nil {
			cmd.report(err)
		}
		return
	}

	for _, path := range args {
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
