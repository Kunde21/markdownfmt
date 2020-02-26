// Package markdown provides a Markdown renderer.
package markdown

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/russross/blackfriday/v2"
)

type markdownRenderer struct {
	normalTextMarker   map[*bytes.Buffer]int
	orderedListCounter map[int]int
	listParagraph      map[int]bool // Used to keep track of whether a given list item uses a paragraph for large spacing.
	listDepth          int          // which depth are we in
	listJustExited     bool         // did we just exited list? to prevent double newline
	blockquoteDepth    int          // how many nested blockquotes are we in
	lastNormalText     string

	// TODO: Clean these up.
	headers      []string
	columnAligns []blackfriday.CellAlignFlags
	columnWidths []int
	cells        []string

	buf *bytes.Buffer
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
func (mr *markdownRenderer) BlockCode(node *blackfriday.Node, lang string) {
	// blockcode fails if in list or blockquote because of blackfriday parser bug
	mr.buf.WriteByte('\n')
	if mr.blockquoteDepth != 0 {
		log.Fatal("Do not use codeblocks inside blockquote, broken in blackfriday")
	}
	if mr.listDepth != 0 {
		log.Fatal("Do not use codeblocks inside list, broken in blackfriday")
	}

	// Parse out the language name.
	count := 0
	for _, elt := range strings.Fields(lang) {
		if elt[0] == '.' {
			elt = elt[1:]
		}
		if len(elt) == 0 {
			continue
		}
		mr.buf.WriteString("```")
		mr.buf.WriteString(elt)
		count++
		break
	}

	if count == 0 {
		mr.buf.WriteString("```")
	}
	mr.buf.WriteString("\n")

	var code []byte
	if formattedCode, ok := formatCode(lang, node.Literal); ok {
		code = formattedCode
	} else {
		code = node.Literal
	}

	mr.buf.Write(bytes.TrimSpace(code))
	mr.buf.WriteString("\n")
	mr.buf.WriteString("```\n")
}

func (mr *markdownRenderer) BlockQuote(node *blackfriday.Node, entering bool) {
	if entering {
		mr.blockquoteDepth++
	} else {
		mr.blockquoteDepth--
	}
}

func (mr *markdownRenderer) BlockHtml(node *blackfriday.Node) {
	mr.buf.WriteByte('\n')
	mr.buf.Write(node.Literal)
}

func (_ *markdownRenderer) stringWidth(s string) int {
	return runewidth.StringWidth(s)
}

func (mr *markdownRenderer) Header(node *blackfriday.Node, text []byte) {
	mr.spaceBeforeParagraph(node)

	if node.Level >= 3 {
		mr.buf.Write(bytes.Repeat([]byte{'#'}, node.Level))
		mr.buf.WriteByte(' ')
	}

	newBuf := bytes.NewBuffer(nil)
	if node.HeadingID != "" {
		fmt.Fprintf(newBuf, " {#%s}", node.HeadingID)
	}
	newBuf.Write(text)
	slen := mr.stringWidth(newBuf.String())

	newBuf.WriteTo(mr.buf)

	switch node.Level {
	case 1:
		mr.buf.WriteByte('\n')
		mr.buf.Write(mr.leader())
		mr.buf.Write(bytes.Repeat([]byte{'='}, slen))
	case 2:
		mr.buf.WriteByte('\n')
		mr.buf.Write(mr.leader())
		mr.buf.Write(bytes.Repeat([]byte{'-'}, slen))
	}

	mr.buf.WriteString("\n")
}

func (mr *markdownRenderer) HRule() {
	mr.buf.WriteString("\n---\n")
}

func (mr *markdownRenderer) List(node *blackfriday.Node, entering bool) {
	if !entering {
		mr.listDepth--
		mr.listJustExited = true
		return
	}
	if mr.blockquoteDepth > 0 {
		log.Fatal("list inside blockquote not supported, sorry")
	}
	mr.listJustExited = false
	if mr.listDepth == 0 || !node.Tight {
		mr.buf.WriteString("\n")
	}
	mr.listDepth++
	if node.ListFlags&blackfriday.ListTypeOrdered != 0 {
		mr.orderedListCounter[mr.listDepth] = 1
	} else {
		mr.orderedListCounter[mr.listDepth] = 0
	}
	mr.listParagraph[mr.listDepth] = !node.Tight
}

// how many spaces to write after item number.
// unfortunately, blackfriday requires the indent to be 4/8/... spaces
// otherwise it breaks in random ways
// blackfriday is not commonmark-compliant; new Hugo already replaced it with new engine
func (mr *markdownRenderer) spacesAfterItem() []byte {
	// it's 0 if it's items
	if mr.orderedListCounter[mr.listDepth] == 0 {
		return bytes.Repeat([]byte{' '}, 3)
	}

	// counter is always 1 bigger
	counter := mr.orderedListCounter[mr.listDepth] - 1

	// let's count how long the string is
	counterString := fmt.Sprintf("%d", counter)
	lenCounter := len(counterString)

	// 4 for "tabs", - length, minus dot
	spaceCount := 4 - (lenCounter + 1)

	return bytes.Repeat([]byte{' '}, spaceCount)
}

// spaces before item starts from start of line
// blackfriday needs each nesting level be at least 4, otherwise it behaves erratically
func (mr *markdownRenderer) spacesBeforeItem(includeLast bool) []byte {
	counter := mr.listDepth * 4
	if !includeLast {
		counter -= 4
	}
	return bytes.Repeat([]byte{' '}, counter)
}

func (mr *markdownRenderer) blockquoteMarks() []byte {
	return blockquotesMarksWithLevel(mr.blockquoteDepth)
}

func blockquotesMarksWithLevel(level int) []byte {
	return bytes.Repeat([]byte{'>', ' '}, level)
}

// leader includes both > and spaces for items
func (mr *markdownRenderer) leader() []byte {
	spaces := mr.spacesBeforeItem(true)
	blockquotes := mr.blockquoteMarks()
	return append(spaces, blockquotes...)
}

// what space to write when we encounter paragraph
// (including the newlines)
func (mr *markdownRenderer) spaceBeforeParagraph(node *blackfriday.Node) {
	isAfterItem := false
	if node.Parent != nil && node.Parent.Type == blackfriday.Item && node.Parent.FirstChild == node {
		// spaces after item treated differently
		mr.buf.Write(mr.spacesAfterItem())
		isAfterItem = true
	}

	// special case - blockquote inside item
	if node.Parent != nil &&
		node.Parent.Parent != nil &&
		node.Type == blackfriday.Paragraph &&
		node.Parent.Type == blackfriday.BlockQuote &&
		node.Parent.Parent.Type == blackfriday.Item &&
		node.Parent.Parent.FirstChild == node.Parent &&
		node.Parent.FirstChild == node {
		mr.buf.Write(mr.spacesAfterItem())
		mr.buf.WriteString("> ")
		isAfterItem = true
	}
	if !isAfterItem {
		mr.newlineBeforeParagraph(node)
		mr.buf.Write(mr.leader())
	}
}

// just newlines before paragraph, when we know we will do a newline
func (mr *markdownRenderer) newlineBeforeParagraph(node *blackfriday.Node) {
	blockquoteDepth := mr.blockquoteDepth
	if node.Parent != nil && node.Parent.Type == blackfriday.BlockQuote && node.Parent.FirstChild == node {
		// space before first blockquote paragraph is with 1 less level
		blockquoteDepth--
	}
	if blockquoteDepth > 0 {
		mr.buf.Write(mr.spacesBeforeItem(true))
		blockquotes := blockquotesMarksWithLevel(blockquoteDepth)
		blockquotes = blockquotes[:len(blockquotes)-1] // remove trailing space on empty paragraph
		mr.buf.Write(blockquotes)
	}

	mr.buf.WriteByte('\n')
}

// recursive function to tell if an item is in the "last branch" of a list
// to prevent multiple repeated newlines after nested list
func isLastNested(node *blackfriday.Node) bool {
	if node == nil {
		return false
	}
	if node.Type == blackfriday.Item {
		if node.ListFlags&blackfriday.ListItemEndOfList != 0 {
			return true
		}
	}
	if node.Parent == nil {
		return false
	}
	if node == node.Parent.LastChild {
		// let's do recursion, it isn't that deep
		return isLastNested(node.Parent)
	}
	return false
}

func (mr *markdownRenderer) item(node *blackfriday.Node, entering bool) {
	if entering {
		mr.buf.Write(mr.blockquoteMarks())
		spaces := mr.spacesBeforeItem(false)
		mr.buf.Write(spaces)
		if node.ListFlags&blackfriday.ListTypeOrdered != 0 {
			s := fmt.Sprintf("%d%s", mr.orderedListCounter[mr.listDepth], string(node.Delimiter))
			mr.buf.WriteString(s)
			mr.orderedListCounter[mr.listDepth]++
		} else {
			mr.buf.WriteByte(node.BulletChar)
		}
	} else {
		if mr.listParagraph[mr.listDepth] {
			if !isLastNested(node) && !mr.listJustExited {
				mr.buf.WriteString("\n")
			}
		}
	}
}

func (mr *markdownRenderer) paragraph(node *blackfriday.Node, entering bool) {
	if entering {
		mr.spaceBeforeParagraph(node)
		return
	}

	mr.buf.WriteString("\n")
}

func (mr *markdownRenderer) table(node *blackfriday.Node, entering bool) {
	if entering {
		mr.spaceBeforeParagraph(node)
		return
	}

	leader := mr.leader()

	for column, cell := range mr.headers {
		mr.buf.WriteByte('|')
		mr.buf.WriteByte(' ')
		mr.buf.WriteString(cell)
		for i := mr.stringWidth(cell); i < mr.columnWidths[column]; i++ {
			mr.buf.WriteByte(' ')
		}
		mr.buf.WriteByte(' ')
	}
	mr.buf.WriteString("|\n")
	mr.buf.Write(leader)
	for column, width := range mr.columnWidths {
		mr.buf.WriteByte('|')
		if mr.columnAligns[column]&blackfriday.TableAlignmentLeft != 0 {
			mr.buf.WriteByte(':')
		} else {
			mr.buf.WriteByte('-')
		}
		for ; width > 0; width-- {
			mr.buf.WriteByte('-')
		}
		if mr.columnAligns[column]&blackfriday.TableAlignmentRight != 0 {
			mr.buf.WriteByte(':')
		} else {
			mr.buf.WriteByte('-')
		}
	}
	mr.buf.WriteString("|\n")
	for i := 0; i < len(mr.cells); {
		mr.buf.Write(leader)
		for column := range mr.headers {
			cell := []byte(mr.cells[i])
			i++
			mr.buf.WriteByte('|')
			mr.buf.WriteByte(' ')
			switch mr.columnAligns[column] {
			default:
				fallthrough
			case blackfriday.TableAlignmentLeft:
				mr.buf.Write(cell)
				for i := mr.stringWidth(string(cell)); i < mr.columnWidths[column]; i++ {
					mr.buf.WriteByte(' ')
				}
			case blackfriday.TableAlignmentCenter:
				spaces := mr.columnWidths[column] - mr.stringWidth(string(cell))
				for i := 0; i < spaces/2; i++ {
					mr.buf.WriteByte(' ')
				}
				mr.buf.Write(cell)
				for i := 0; i < spaces-(spaces/2); i++ {
					mr.buf.WriteByte(' ')
				}
			case blackfriday.TableAlignmentRight:
				for i := mr.stringWidth(string(cell)); i < mr.columnWidths[column]; i++ {
					mr.buf.WriteByte(' ')
				}
				mr.buf.Write(cell)
			}
			mr.buf.WriteByte(' ')
		}
		mr.buf.WriteString("|\n")
	}

	mr.headers = nil
	mr.columnAligns = nil
	mr.columnWidths = nil
	mr.cells = nil
}

func (mr *markdownRenderer) tableHeaderCell(text []byte, align blackfriday.CellAlignFlags) {
	mr.columnAligns = append(mr.columnAligns, align)
	columnWidth := mr.stringWidth(string(text))
	mr.columnWidths = append(mr.columnWidths, columnWidth)
	mr.headers = append(mr.headers, string(text))
}

func (mr *markdownRenderer) tableCell(text []byte, align blackfriday.CellAlignFlags) {
	columnWidth := mr.stringWidth(string(text))
	column := len(mr.cells) % len(mr.headers)
	if columnWidth > mr.columnWidths[column] {
		mr.columnWidths[column] = columnWidth
	}
	mr.cells = append(mr.cells, string(text))
}

// Span-level callbacks.

func (mr *markdownRenderer) codeSpan(text []byte) {
	mr.buf.WriteByte('`')
	mr.buf.Write(text)
	mr.buf.WriteByte('`')
}

func (mr *markdownRenderer) doubleEmphasis(content []byte) {
	if len(content) == 0 {
		return
	}
	mr.buf.WriteString("**")
	mr.buf.Write(content)
	mr.buf.WriteString("**")
}

func (mr *markdownRenderer) emphasis(content []byte) {
	if len(content) == 0 {
		return
	}
	mr.buf.WriteByte('*')
	mr.buf.Write(content)
	mr.buf.WriteByte('*')
}

func (mr *markdownRenderer) image(link []byte, title []byte, alt []byte) {
	mr.buf.WriteString("![")
	mr.buf.Write(alt)
	mr.buf.WriteString("](")
	mr.buf.Write(escape(link))
	if len(title) != 0 {
		mr.buf.WriteString(` "`)
		mr.buf.Write(title)
		mr.buf.WriteString(`"`)
	}
	mr.buf.WriteString(")")
}

func (mr *markdownRenderer) lineBreak() {
	mr.buf.WriteString("  \n")

	spaces := mr.leader()
	mr.buf.Write(spaces)
}

func (mr *markdownRenderer) link(link []byte, title []byte, content []byte) {
	mr.buf.WriteString("[")
	mr.buf.Write(content)
	mr.buf.WriteString("](")
	mr.buf.Write(escape(link))
	if len(title) != 0 {
		mr.buf.WriteString(` "`)
		mr.buf.Write(title)
		mr.buf.WriteString(`"`)
	}
	mr.buf.WriteString(")")
}

func (mr *markdownRenderer) rawHtmlTag(node *blackfriday.Node) {
	mr.buf.Write(node.Literal)
}

func (mr *markdownRenderer) strikeThrough(content []byte) {
	if len(content) == 0 {
		return
	}
	mr.buf.WriteString("~~")
	mr.buf.Write(content)
	mr.buf.WriteString("~~")
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

func (mr *markdownRenderer) NormalText(node *blackfriday.Node) {
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
	if mr.skipSpaceIfNeededNormalText(cleanString) { // Skip first space if last character is already a space (i.e., no need for a 2nd space in a row).
		cleanString = cleanString[1:]
	}

	mr.buf.WriteString(cleanString)
	if len(cleanString) >= 1 && cleanString[len(cleanString)-1] == ' ' { // If it ends with a space, make note of that.
		mr.normalTextMarker[mr.buf] = mr.buf.Len()
	}
}

// Header and footer.
func (_ *markdownRenderer) RenderHeader(io.Writer, *blackfriday.Node) {}
func (_ *markdownRenderer) RenderFooter(io.Writer, *blackfriday.Node) {}

func (mr *markdownRenderer) skipSpaceIfNeededNormalText(cleanString string) bool {
	if cleanString[0] != ' ' {
		return false
	}
	if _, ok := mr.normalTextMarker[mr.buf]; !ok {
		mr.normalTextMarker[mr.buf] = -1
	}
	return mr.normalTextMarker[mr.buf] == mr.buf.Len()
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

// NewRenderer returns a Markdown renderer.
// If opt is nil the defaults are used.
func NewRenderer() blackfriday.Renderer {
	mr := &markdownRenderer{
		normalTextMarker:   make(map[*bytes.Buffer]int),
		orderedListCounter: make(map[int]int),
		listParagraph:      make(map[int]bool),

		buf: bytes.NewBuffer(nil),
	}
	return mr
}

// Process formats Markdown.
// If opt is nil the defaults are used.
// Error can only occur when reading input from filename rather than src.
func Process(filename string, src []byte) ([]byte, error) {
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
		blackfriday.WithRenderer(NewRenderer()),
		blackfriday.WithExtensions(extensions),
	)
	// cuts newline because we sometimes output more newlines
	return bytes.TrimLeft(output, "\n"), nil
}

// If src != nil, readSource returns src.
// If src == nil, readSource returns the result of reading the file specified by filename.
func readSource(filename string, src []byte) ([]byte, error) {
	if src != nil {
		return src, nil
	}
	return ioutil.ReadFile(filename)
}
