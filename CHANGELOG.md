# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v3.0.0 - 2022-12-14

This is a new major release. You can install it using the module path: github.com/Kunde21/markdownfmt/v3.

Note that we moved some packages in this release. The import paths for different components are now:

| Component   | Old import path                               | New import path                                   |
|-------------|-----------------------------------------------|---------------------------------------------------|
| CLI         | github.com/Kunde21/markdownfmt/v2             | github.com/Kunde21/markdownfmt/v3/cmd/markdownfmt |
| markdownfmt | github.com/Kunde21/markdownfmt/v2/markdownfmt | github.com/Kunde21/markdownfmt/v3                 |
| markdown    | github.com/Kunde21/markdownfmt/v2/markdown    | github.com/Kunde21/markdownfmt/v3/markdown        |

### Added
- Support raw HTML blocks.
- Add `WithSoftWraps` to retain soft line breaks.
- Add `WithCodeFormatters` to supply custom formatters for code blocks, and a `GoCodeFormatter` built-in formatter.
- Add `WithEmphasisToken` and `WithStrongToken` to change the tokens used for bold and italic text.
- `markdownfmt` CLI: Add `-gofmt` flag to enable reformatting of Go source code.

### Removed
- Deleted `markdownfmt.NewParser`. If you need this, use `markdownfmt.NewGoldmark` to get a `goldmark.Markdown` and extract the parser from that.

### Changed
- Move `markdownfmt` CLI to cmd/markdownfmt.
- Move `markdownfmt` package to module root.
- Change module import path to github.com/Kunde21/markdownfmt/v3.
- Don't modify code inside fenced code by default. Supply the `WithCodeFormatters` option to the `Renderer` to enable reformatting of source code.
- `markdownfmt` CLI: Don't shell out to `diff` in `-d` mode.
- `markdownfmt` CLI: Don't reformat Go source code inside fenced code blocks. Opt into this functionality with the `-gofmt` flag.
- `Renderer.AddOptions` is no longer a no-op. It now extracts and applies Markdown-specific options from the provided list.

### Fixed
- Fix formatting of whitespace in code blocks.
- Retain start positions for ordered lists.
- Significant performance improvements to rendering.

## 2.1.0 - 2021-01-26

### Added
- Support autolinks, task check lists, and attribute lists.
- Add opt-in for setext-style headers (`===`, `---`).

### Fixed
- Ignore errors in reformatting Go code.

## 2.0.3 - 2020-10-28

### Changed
- Use ATX-style (`#`) headers only.

### Fixed
- Support multiple words for info strings in fenced code blocks.

## 2.0.1 - 2020-03-05

### Fixed
- Fix panic on code using unsupported Markdown extensions.

## 2.0.0 - 2020-03-03

- Initial release using goldmark instead of blackfriday.
