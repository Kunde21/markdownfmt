# markdownfmt

[![Build Status](https://github.com/Kunde21/markdownfmt/actions/workflows/go.yml/badge.svg?query=branch%3Amaster)](https://github.com/Kunde21/markdownfmt/actions/workflows/go.yml?query=branch%3Amaster) [![Go Reference](https://pkg.go.dev/badge/github.com/Kunde21/markdownfmt/v3.svg)](https://pkg.go.dev/github.com/Kunde21/markdownfmt/v3)

markdownfmt is a CLI that reformats Markdown files (like `gofmt` but for Markdown) and a library that you can use to generate well-formed Markdown files.

**Features**

- Full [GitHub Flavored markdown](https://github.github.com/gfm) support
- Fenced Code Blocks with longer info strings (see [shurcooL#58](https://github.com/shurcooL/markdownfmt/issues/58))
- ATX-style headers (`#`, `##`) by default

## Installation

### CLI

```bash
go install github.com/Kunde21/markdownfmt/v3@latest
```

### Library

```bash
go get github.com/Kunde21/markdownfmt/v3@latest
```

## Usage

```
usage: markdownfmt [flags] [path ...]
  -d    display diffs instead of rewriting files
  -gofmt
        reformat Go source inside fenced code blocks
  -l    list files whose formatting differs from markdownfmt's
  -soft-wraps
        wrap lines even on soft line breaks
  -u    write underline headings instead of hashes for levels 1 and 2
  -w    write result to (source) file instead of stdout
```

The markdownfmt CLI supports the following execution modes:

* stdout: Write reformatted contents of provided files to stdout. This is the default.
* write (`-w`): Reformat and rewrite Markdown files in-place.
* list (`-l`): List files that would be modified, but don't change them.
* diff (`-d`): Display a diff of modifications that would be made to files, but don't change them.

## History

markdownfmt began as a fork of [shurcooL/markdownfmt](https://github.com/shurcooL/markdownfmt) targeting [Goldmark](https://github.com/yuin/goldmark) instead of [Blackfriday](https://github.com/russross/blackfriday). It has since diverged significantly.

## Related projects

* [shurcooL/markdownfmt](https://github.com/shurcooL/markdownfmt): The project that this forked from.
* [mdox](https://github.com/bwplotka/mdox/): Builds upon markdownfmt. Adds support for link validation, command execution, and more.

### Editor Plugins

- [vim-markdownfmt](https://github.com/moorereason/vim-markdownfmt) for Vim.
- [emacs-markdownfmt](https://github.com/nlamirault/emacs-markdownfmt) for Emacs.
- Built-in in Conception.
- [markdown-format](https://atom.io/packages/markdown-format) for Atom (deprecated).
- Add a plugin for your favorite editor here?

### Alternatives

- [`mdfmt`](https://github.com/moorereason/mdfmt) - Fork of `markdownfmt` that adds front matter support.
- [`tidy-markdown`](https://github.com/slang800/tidy-markdown) - Project with similar goals, but written in JS and based on a slightly different [styleguide](https://github.com/slang800/markdown-styleguide).

## License

[MIT License](https://opensource.org/licenses/mit-license.php)
