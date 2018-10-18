// Package markdown provides a Markdown renderer.
package markdown

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
	blackfriday "github.com/russross/blackfriday/v2"
	"github.com/shurcooL/go/indentwriter"
)

type markdownRenderer struct {
	normalTextMarker   map[*bytes.Buffer]int
	orderedListCounter map[int]int
	listParagraph      map[int]bool // Used to keep track of whether a given list item uses a paragraph for large spacing.
	listDepth          int
	lastNormalText     string

	// TODO: Clean these up.
	headers      []string
	columnAligns []blackfriday.CellAlignFlags
	columnWidths []int
	cells        []string

	opt    Options
	leader [][]byte

	buf *bytes.Buffer

	// stringWidth is used internally to calculate visual width of a string.
	stringWidth func(s string) (width int)
}

func formatCode(lang string, text []byte) (formattedCode []byte, ok bool) {
	switch lang {
	case "Go", "go":
		gofmt, err := format.Source(text)
		if err != nil {
			return nil, false
		}
		return gofmt, true
	default:
		return nil, false
	}
}

// Block-level callbacks.
func (mr *markdownRenderer) BlockCode(out *bytes.Buffer, node *blackfriday.Node, lang string) {
	mr.doubleSpace(out)

	// Parse out the language name.
	count := 0
	for _, elt := range strings.Fields(lang) {
		if elt[0] == '.' {
			elt = elt[1:]
		}
		if len(elt) == 0 {
			continue
		}
		out.WriteString("```")
		out.WriteString(elt)
		count++
		break
	}

	if count == 0 {
		out.WriteString("```")
	}
	out.WriteString("\n")

	if formattedCode, ok := formatCode(lang, node.Literal); ok {
		out.Write(bytes.TrimSpace(formattedCode))
	} else {
		out.Write(bytes.TrimSpace(node.Literal))
	}

	out.WriteString("\n```")
}

func (mr *markdownRenderer) BlockQuote(out *bytes.Buffer, node *blackfriday.Node, entering bool) {
	text := node.Literal
	mr.doubleSpace(out)
	lines := bytes.Split(text, []byte("\n"))
	for i, line := range lines {
		if i == len(lines)-1 {
			continue
		}
		out.WriteString(">")
		if len(line) != 0 {
			out.WriteString(" ")
			out.Write(line)
		}
		out.WriteString("\n")
	}
	if entering {
		mr.leader = append(mr.leader, []byte("> "))
	} else {
		mr.leader = mr.leader[:len(mr.leader)-1]
	}
}

func (mr *markdownRenderer) BlockHtml(node *blackfriday.Node) {
	mr.buf.WriteByte('\n')
	mr.buf.Write(node.Literal)
}

func (_ *markdownRenderer) TitleBlock(out *bytes.Buffer, text []byte) {}

func (mr *markdownRenderer) Header(node *blackfriday.Node, entering bool) {
	if entering {
		mr.doubleSpace(nil)
		if node.IsTitleblock && node.Level >= 3 {
			mr.leader = append(mr.leader, append(bytes.Repeat([]byte{'#'}, node.Level), ' '))
		} else if !node.IsTitleblock {
			mr.leader = append(mr.leader, append(bytes.Repeat([]byte{'#'}, node.Level), ' '))
		}
		return
	}

	if node.HeadingID != "" {
		fmt.Fprintf(mr.buf, " {#%s}", node.HeadingID)
	}
	if node.IsTitleblock {
		len := mr.stringWidth(mr.buf.String())
		switch node.Level {
		case 1:
			fmt.Fprint(mr.buf, "\n", strings.Repeat("=", len))
		case 2:
			fmt.Fprint(mr.buf, "\n", strings.Repeat("-", len))
		}
	}
	mr.leader = mr.leader[:len(mr.leader)-1]
	mr.buf.WriteString("\n")
}

func (mr *markdownRenderer) HRule() {
	mr.buf.WriteString("\n---")
}

func (mr *markdownRenderer) List(out *bytes.Buffer, node *blackfriday.Node, entering bool) {
	if !entering {
		mr.listDepth--
		return
	}

	mr.listDepth++
	if node.ListFlags&blackfriday.ListTypeOrdered != 0 {
		mr.orderedListCounter[mr.listDepth] = 1
	}
	mr.listParagraph[mr.listDepth] = !node.Tight
}

func (mr *markdownRenderer) item(out *bytes.Buffer, node *blackfriday.Node, entering bool) {
	if entering {
		out.WriteString("\n")
		if node.ListFlags&blackfriday.ListTypeOrdered != 0 {
			fmt.Fprintf(out, "%d%v", mr.orderedListCounter[mr.listDepth], node.Delimiter)
			indentwriter.New(out, mr.listDepth-1).Write(node.Literal)
			mr.orderedListCounter[mr.listDepth]++
		} else {
			indentwriter.New(out, mr.listDepth-1).Write([]byte{node.BulletChar})
		}
	} else {
		if mr.listParagraph[mr.listDepth] {
			if node.ListFlags&blackfriday.ListItemEndOfList == 0 {
				out.WriteString("EOL\n")
			}
		}
	}
}

func (mr *markdownRenderer) paragraph(out *bytes.Buffer, entering bool) {
	//text := node.Literal
	if !mr.listParagraph[mr.listDepth] && mr.listDepth != 0 {
		return
	}
	if entering {
		mr.doubleSpace(out)
	} else {
		out.WriteString("\n")
	}
}

func (mr *markdownRenderer) table(out *bytes.Buffer, node *blackfriday.Node, entering bool) {
	if entering {
		mr.doubleSpace(out)
		return
	}
	for column, cell := range mr.headers {
		out.WriteByte('|')
		out.WriteByte(' ')
		out.WriteString(cell)
		for i := mr.stringWidth(cell); i < mr.columnWidths[column]; i++ {
			out.WriteByte(' ')
		}
		out.WriteByte(' ')
	}
	out.WriteString("|\n")
	for column, width := range mr.columnWidths {
		out.WriteByte('|')
		if mr.columnAligns[column]&blackfriday.TableAlignmentLeft != 0 {
			out.WriteByte(':')
		} else {
			out.WriteByte('-')
		}
		for ; width > 0; width-- {
			out.WriteByte('-')
		}
		if mr.columnAligns[column]&blackfriday.TableAlignmentRight != 0 {
			out.WriteByte(':')
		} else {
			out.WriteByte('-')
		}
	}
	out.WriteString("|\n")
	for i := 0; i < len(mr.cells); {
		for column := range mr.headers {
			cell := []byte(mr.cells[i])
			i++
			out.WriteByte('|')
			out.WriteByte(' ')
			switch mr.columnAligns[column] {
			default:
				fallthrough
			case blackfriday.TableAlignmentLeft:
				out.Write(cell)
				for i := mr.stringWidth(string(cell)); i < mr.columnWidths[column]; i++ {
					out.WriteByte(' ')
				}
			case blackfriday.TableAlignmentCenter:
				spaces := mr.columnWidths[column] - mr.stringWidth(string(cell))
				for i := 0; i < spaces/2; i++ {
					out.WriteByte(' ')
				}
				out.Write(cell)
				for i := 0; i < spaces-(spaces/2); i++ {
					out.WriteByte(' ')
				}
			case blackfriday.TableAlignmentRight:
				for i := mr.stringWidth(string(cell)); i < mr.columnWidths[column]; i++ {
					out.WriteByte(' ')
				}
				out.Write(cell)
			}
			out.WriteByte(' ')
		}
		out.WriteString("|\n")
	}

	mr.headers = nil
	mr.columnAligns = nil
	mr.columnWidths = nil
	mr.cells = nil
}

func (mr *markdownRenderer) tableHeaderCell(out *bytes.Buffer, text []byte, align blackfriday.CellAlignFlags) {
	//text := node.Literal
	mr.columnAligns = append(mr.columnAligns, align)
	columnWidth := mr.stringWidth(string(text))
	mr.columnWidths = append(mr.columnWidths, columnWidth)
	mr.headers = append(mr.headers, string(text))
}

func (mr *markdownRenderer) tableCell(out *bytes.Buffer, text []byte, align blackfriday.CellAlignFlags) {
	//text := node.Literal
	columnWidth := mr.stringWidth(string(text))
	column := len(mr.cells) % len(mr.headers)
	if columnWidth > mr.columnWidths[column] {
		mr.columnWidths[column] = columnWidth
	}
	mr.cells = append(mr.cells, string(text))
}

func (_ *markdownRenderer) footnotes(out *bytes.Buffer, text func() bool) {
	out.WriteString("<Footnotes: Not implemented.>") // TODO
}

func (_ *markdownRenderer) footnoteItem(out *bytes.Buffer, name, text []byte, flags int) {
	out.WriteString("<FootnoteItem: Not implemented.>") // TODO
}

// Span-level callbacks.
func (_ *markdownRenderer) autoLink(out *bytes.Buffer, link []byte, kind int) {
	//text := node.Literal
	out.Write(escape(link))
}

func (_ *markdownRenderer) codeSpan(out *bytes.Buffer, text []byte) {
	//text := node.Literal
	out.WriteByte('`')
	out.Write(text)
	out.WriteByte('`')
}

func (mr *markdownRenderer) doubleEmphasis(out *bytes.Buffer, text []byte) {
	//text := node.Literal
	if mr.opt.Terminal {
		out.WriteString("\x1b[1m") // Bold.
	}
	out.WriteString("**")
	out.Write(text)
	out.WriteString("**")
	if mr.opt.Terminal {
		out.WriteString("\x1b[0m") // Reset.
	}
}

func (_ *markdownRenderer) emphasis(out *bytes.Buffer, text []byte) {
	//text := node.Literal
	if len(text) == 0 {
		return
	}
	out.WriteByte('*')
	out.Write(text)
	out.WriteByte('*')
}

func (_ *markdownRenderer) image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	//text := node.Literal
	out.WriteString("![")
	out.Write(alt)
	out.WriteString("](")
	out.Write(escape(link))
	if len(title) != 0 {
		out.WriteString(` "`)
		out.Write(title)
		out.WriteString(`"`)
	}
	out.WriteString(")")
}

func (_ *markdownRenderer) lineBreak(out *bytes.Buffer) {
	out.WriteString("  \n")
}

func (_ *markdownRenderer) link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	//text := node.Literal
	out.WriteString("[")
	out.Write(content)
	out.WriteString("](")
	out.Write(escape(link))
	if len(title) != 0 {
		out.WriteString(` "`)
		out.Write(title)
		out.WriteString(`"`)
	}
	out.WriteString(")")
}

func (mr *markdownRenderer) rawHtmlTag(node *blackfriday.Node) {
	mr.buf.Write(node.Literal)
}

func (_ *markdownRenderer) tripleEmphasis(out *bytes.Buffer, text []byte) {
	//text := node.Literal
	out.WriteString("***")
	out.Write(text)
	out.WriteString("***")
}

func (_ *markdownRenderer) StrikeThrough(out *bytes.Buffer, text []byte) {
	//text := node.Literal
	out.WriteString("~~")
	out.Write(text)
	out.WriteString("~~")
}

func (_ *markdownRenderer) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
	out.WriteString("<FootnoteRef: Not implemented.>") // TODO
}

// escape replaces instances of backslash with escaped backslash in text.
func escape(text []byte) []byte {
	return bytes.Replace(text, []byte(`\`), []byte(`\\`), -1)
}

func isNumber(data []byte) bool {
	for _, b := range data {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func needsEscaping(text []byte, lastNormalText string) bool {
	switch string(text) {
	case `\`,
		"`",
		"*",
		"_",
		"{", "}",
		"[", "]",
		"(", ")",
		"#",
		"+",
		"-":
		return true
	case "!":
		return false
	case ".":
		// Return true if number, because a period after a number must be escaped to not get parsed as an ordered list.
		return isNumber([]byte(lastNormalText))
	case "<", ">":
		return true
	default:
		return false
	}
}

// Low-level callbacks.
func (_ *markdownRenderer) Entity(out *bytes.Buffer, entity []byte) {
	//text := node.Literal
	out.Write(entity)
}
func (mr *markdownRenderer) NormalText(out *bytes.Buffer, node *blackfriday.Node) {
	text := node.Literal
	normalText := string(text)
	if needsEscaping(text, mr.lastNormalText) {
		text = append([]byte("\\"), text...)
	}
	mr.lastNormalText = normalText
	if mr.listDepth > 0 && string(text) == "\n" { // TODO: See if this can be cleaned up... It's needed for lists.
		return
	}
	cleanString := cleanWithoutTrim(string(text))
	if cleanString == "" {
		return
	}
	if mr.skipSpaceIfNeededNormalText(out, cleanString) { // Skip first space if last character is already a space (i.e., no need for a 2nd space in a row).
		cleanString = cleanString[1:]
	}
	out.Write(bytes.Join(mr.leader, []byte{}))
	out.WriteString(cleanString)
	if len(cleanString) >= 1 && cleanString[len(cleanString)-1] == ' ' { // If it ends with a space, make note of that.
		mr.normalTextMarker[out] = out.Len()
	}
}

// Header and footer.
func (_ *markdownRenderer) RenderHeader(io.Writer, *blackfriday.Node) {}
func (_ *markdownRenderer) RenderFooter(io.Writer, *blackfriday.Node) {}

func (_ *markdownRenderer) GetFlags() int { return 0 }

func (mr *markdownRenderer) skipSpaceIfNeededNormalText(out *bytes.Buffer, cleanString string) bool {
	if cleanString[0] != ' ' {
		return false
	}
	if _, ok := mr.normalTextMarker[out]; !ok {
		mr.normalTextMarker[out] = -1
	}
	return mr.normalTextMarker[out] == out.Len()
}

// cleanWithoutTrim is like clean, but doesn't trim blanks.
func cleanWithoutTrim(s string) string {
	var b []byte
	var p byte
	for i := 0; i < len(s); i++ {
		q := s[i]
		if q == '\n' || q == '\r' || q == '\t' {
			q = ' '
		}
		if q != ' ' || p != ' ' {
			b = append(b, q)
			p = q
		}
	}
	return string(b)
}

func (mr *markdownRenderer) doubleSpace(out *bytes.Buffer) {
	mr.buf.WriteByte('\n')
}

// terminalStringWidth returns width of s, taking into account possible ANSI escape codes
// (which don't count towards string width).
func terminalStringWidth(s string) (width int) {
	width = runewidth.StringWidth(s)
	width -= strings.Count(s, "\x1b[1m") * len("[1m") // HACK, TODO: Find a better way of doing this.
	width -= strings.Count(s, "\x1b[0m") * len("[0m") // HACK, TODO: Find a better way of doing this.
	return width
}

// NewRenderer returns a Markdown renderer.
// If opt is nil the defaults are used.
func NewRenderer(opt *Options) blackfriday.Renderer {
	mr := &markdownRenderer{
		normalTextMarker:   make(map[*bytes.Buffer]int),
		orderedListCounter: make(map[int]int),
		listParagraph:      make(map[int]bool),

		buf: bytes.NewBuffer(nil),

		stringWidth: runewidth.StringWidth,
	}
	if opt != nil {
		mr.opt = *opt
	}
	if mr.opt.Terminal {
		mr.stringWidth = terminalStringWidth
	}
	return mr
}

// Options specifies options for formatting.
type Options struct {
	// Terminal specifies if ANSI escape codes are emitted for styling.
	Terminal bool
}

// Process formats Markdown.
// If opt is nil the defaults are used.
// Error can only occur when reading input from filename rather than src.
func Process(filename string, src []byte, opt *Options) ([]byte, error) {
	// Get source.
	text, err := readSource(filename, src)
	if err != nil {
		return nil, err
	}

	// extensions for GitHub Flavored Markdown-like parsing.
	const extensions = blackfriday.NoIntraEmphasis |
		blackfriday.Tables |
		blackfriday.FencedCode |
		blackfriday.Autolink |
		blackfriday.Strikethrough |
		blackfriday.SpaceHeadings |
		blackfriday.NoEmptyLineBeforeBlock

	// output := blackfriday.Markdown(text, NewRenderer(opt), extensions)
	output := blackfriday.Run(text,
		blackfriday.WithRenderer(NewRenderer(opt)),
		blackfriday.WithExtensions(extensions),
	)
	return output, nil
}

// If src != nil, readSource returns src.
// If src == nil, readSource returns the result of reading the file specified by filename.
func readSource(filename string, src []byte) ([]byte, error) {
	if src != nil {
		return src, nil
	}
	return ioutil.ReadFile(filename)
}
