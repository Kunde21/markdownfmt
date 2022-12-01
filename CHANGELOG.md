# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Added
- Support raw HTML blocks.
- Add `WithSoftWraps` to retain soft line breaks.
- Add `WithCodeFormatters` to supply custom formatters for code blocks.
- Add `WithEmphasisToken` and `WithStrongToken` to change the tokens used for bold and italic text.

### Changed
- The `markdownfmt` CLI no longer shells out to `diff`.

### Fixed
- Fix formatting of whitespace in code blocks.
- Retain start positions for ordered lists.

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
