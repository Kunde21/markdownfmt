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

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extAst "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"

	"github.com/mattn/go-runewidth"
)

type MarkdownFmtRenderer struct {
	normalTextMarker   map[*bytes.Buffer]int
	orderedListCounter map[int]int
	listParagraph      map[int]bool // Used to keep track of whether a given list item uses a paragraph for large spacing.
	listDepth          int          // which depth are we in
	listJustExited     bool         // did we just exited list? to prevent double newline
	blockquoteDepth    int          // how many nested blockquotes are we in
	lastNormalText     string

	// TODO: Clean these up.
	headers      []string
	columnAligns []extAst.Alignment
	columnWidths []int
	cells        []string

	tableIsHeader bool

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

func rawWrite(writer io.Writer, source []byte) {
	n := 0
	l := len(source)
	for i := 0; i < l; i++ {
		v := source[i]
		_, _ = writer.Write(source[i-n : i])
		n = 0
		_, _ = writer.Write([]byte{v})
	}
	if n != 0 {
		_, _ = writer.Write(source[l-n:])
	}
}

func writeLines(w io.Writer, source []byte, n ast.Node) {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		rawWrite(w, line.Value(source))
	}
}

// Block-level callbacks.
func (mr *MarkdownFmtRenderer) blockCode(node ast.Node, source []byte) {
	mr.spaceBeforeParagraph(node)

	lang := ""
	tnode, isFenced := node.(*ast.FencedCodeBlock)
	mr.buf.WriteString("```")
	if isFenced && tnode.Info != nil {
		mr.buf.Write(tnode.Info.Text(source))

		for _, elt := range strings.Fields(lang) {
			elt = strings.TrimSpace(strings.TrimLeft(elt, ". "))
			if len(elt) == 0 {
				continue
			}
			lang = elt
			break
		}
	}

	mr.buf.WriteString("\n")
	mr.buf.Write(mr.leader())

	codeBuf := bytes.NewBuffer(nil)
	writeLines(codeBuf, source, node)
	literal := codeBuf.Bytes()

	var code []byte
	if formattedCode, ok := formatCode(lang, literal); ok {
		code = formattedCode
	} else {
		code = literal
	}
	code = bytes.TrimSpace(code)
	code = bytes.ReplaceAll(code, []byte{'\n'}, append([]byte{'\n'}, mr.leader()...))

	mr.buf.Write(code)
	mr.buf.WriteString("\n")
	mr.buf.Write(mr.leader())
	mr.buf.WriteString("```\n")
}

func (mr *MarkdownFmtRenderer) blockQuote(entering bool) {
	if entering {
		mr.blockquoteDepth++
	} else {
		mr.blockquoteDepth--
	}
}

func (mr *MarkdownFmtRenderer) rawHtml(node *ast.RawHTML, source []byte) {
	l := node.Segments.Len()
	for i := 0; i < l; i++ {
		segment := node.Segments.At(i)
		_, _ = mr.buf.Write(segment.Value(source))
	}
}

func (mr *MarkdownFmtRenderer) blockHtml(node *ast.HTMLBlock, source []byte) {
	mr.buf.WriteByte('\n')

	l := node.Lines().Len()
	for i := 0; i < l; i++ {
		line := node.Lines().At(i)
		_, _ = mr.buf.Write(line.Value(source))
	}
}

func (_ *MarkdownFmtRenderer) stringWidth(s string) int {
	return runewidth.StringWidth(s)
}

func (mr *MarkdownFmtRenderer) header(node *ast.Heading, text []byte) {
	mr.spaceBeforeParagraph(node)

	mr.buf.Write(bytes.Repeat([]byte{'#'}, node.Level))
	mr.buf.WriteByte(' ')

	newBuf := bytes.NewBuffer(nil)

	newBuf.Write(text)
	id, hasId := node.AttributeString("id")
	if hasId {
		fmt.Fprintf(newBuf, " {#%s}", id)
	}

	mr.buf.Write(newBuf.Bytes())
	mr.buf.WriteString("\n")
}

func (mr *MarkdownFmtRenderer) hrule() {
	mr.buf.WriteString("\n---\n")
}

func (mr *MarkdownFmtRenderer) list(node *ast.List, entering bool) {
	if !entering {
		mr.listDepth--
		mr.listJustExited = true
		return
	}
	if mr.blockquoteDepth > 0 {
		// not supported because I am lazy, not for any other reason
		// I just suppose that if blockquote and list at same time -> list first
		// FIXME: will need redesign of blockquoteDepth/list handling if we really want this
		log.Fatal("list inside blockquote not supported, sorry")
	}
	mr.listJustExited = false
	if mr.listDepth == 0 || !node.IsTight {
		mr.buf.WriteString("\n")
	}
	mr.listDepth++
	if node.IsOrdered() {
		mr.orderedListCounter[mr.listDepth] = 1
	} else {
		mr.orderedListCounter[mr.listDepth] = 0
	}
	mr.listParagraph[mr.listDepth] = !node.IsTight
}

// how many spaces to write after item number.
func (mr *MarkdownFmtRenderer) spacesAfterItem() []byte {
	return []byte{' '}
}

// spaces before item starts from start of line
func (mr *MarkdownFmtRenderer) spacesBeforeItem(includeLast bool) []byte {
	max := mr.listDepth
	if !includeLast {
		max--
	}
	spaceCount := 0
	for i := 1; i <= max; i++ {
		if mr.orderedListCounter[i] == 0 {
			spaceCount += 2
		} else {
			// counter is always 1 bigger
			counter := mr.orderedListCounter[i] - 1
			// let's count how long the string is
			counterString := fmt.Sprintf("%d", counter)
			lenCounter := len(counterString)
			spaceCount += lenCounter
			spaceCount += 2
		}
	}
	return bytes.Repeat([]byte{' '}, spaceCount)
}

func (mr *MarkdownFmtRenderer) blockquoteMarks() []byte {
	return blockquotesMarksWithLevel(mr.blockquoteDepth)
}

func blockquotesMarksWithLevel(level int) []byte {
	return bytes.Repeat([]byte{'>', ' '}, level)
}

// leader includes both > and spaces for items
func (mr *MarkdownFmtRenderer) leader() []byte {
	spaces := mr.spacesBeforeItem(true)
	blockquotes := mr.blockquoteMarks()
	return append(spaces, blockquotes...)
}

// what space to write when we encounter paragraph
// (including the newlines)
func (mr *MarkdownFmtRenderer) spaceBeforeParagraph(node ast.Node) {
	isAfterItem := false
	par := node.Parent()
	if par != nil && par.Kind() == ast.KindListItem && par.FirstChild() == node {
		// spaces after item treated differently
		mr.buf.Write(mr.spacesAfterItem())
		isAfterItem = true
	}

	var gpar ast.Node
	if par != nil {
		gpar = par.Parent()
	}

	// special case - blockquote inside item
	if gpar != nil &&
		node.Kind() == ast.KindParagraph &&
		par.Kind() == ast.KindBlockquote &&
		gpar.Kind() == ast.KindListItem &&
		gpar.FirstChild() == par &&
		par.FirstChild() == node {
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
func (mr *MarkdownFmtRenderer) newlineBeforeParagraph(node ast.Node) {
	blockquoteDepth := mr.blockquoteDepth
	par := node.Parent()
	if par != nil && par.Kind() == ast.KindBlockquote && par.FirstChild() == node {
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
func isLastNested(node ast.Node) bool {
	if node == nil {
		return false
	}
	par := node.Parent()
	if par == nil {
		return false
	}
	if node.Kind() == ast.KindList && par.Kind() != ast.KindListItem {
		return true
	}

	if par.LastChild() == node {
		// let's do recursion, it isn't that deep
		return isLastNested(par)
	}

	return false
}

func (mr *MarkdownFmtRenderer) item(node *ast.ListItem, entering bool, source []byte) {
	parList := node.Parent().(*ast.List)
	marker := parList.Marker
	if entering {
		mr.buf.Write(mr.blockquoteMarks())
		spaces := mr.spacesBeforeItem(false)
		mr.buf.Write(spaces)
		if parList.IsOrdered() {
			s := fmt.Sprintf("%d%c", mr.orderedListCounter[mr.listDepth], marker)
			mr.buf.WriteString(s)
			mr.orderedListCounter[mr.listDepth]++
		} else {
			mr.buf.WriteByte(marker)
		}
	} else if mr.listParagraph[mr.listDepth] {
		if !isLastNested(node) && !mr.listJustExited {
			mr.buf.WriteString("\n")
		}
	}
}

func (mr *MarkdownFmtRenderer) paragraph(node ast.Node, entering bool) {
	if entering {
		mr.spaceBeforeParagraph(node)
		return
	}

	mr.buf.WriteString("\n")
}

func (mr *MarkdownFmtRenderer) tableHeaderCell(text []byte, align extAst.Alignment) {
	mr.columnAligns = append(mr.columnAligns, align)
	columnWidth := mr.stringWidth(string(text))
	mr.columnWidths = append(mr.columnWidths, columnWidth)
	mr.headers = append(mr.headers, string(text))
}

func (mr *MarkdownFmtRenderer) tableCell(text []byte) {
	columnWidth := mr.stringWidth(string(text))
	column := len(mr.cells) % len(mr.headers)
	if columnWidth > mr.columnWidths[column] {
		mr.columnWidths[column] = columnWidth
	}
	mr.cells = append(mr.cells, string(text))
}

func (mr *MarkdownFmtRenderer) table(node *extAst.Table, entering bool) {
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
		if mr.columnAligns[column] == extAst.AlignLeft ||
			mr.columnAligns[column] == extAst.AlignCenter {
			mr.buf.WriteByte(':')
		} else {
			mr.buf.WriteByte('-')
		}
		for ; width > 0; width-- {
			mr.buf.WriteByte('-')
		}
		if mr.columnAligns[column] == extAst.AlignRight ||
			mr.columnAligns[column] == extAst.AlignCenter {
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
			case extAst.AlignLeft:
				mr.buf.Write(cell)
				for i := mr.stringWidth(string(cell)); i < mr.columnWidths[column]; i++ {
					mr.buf.WriteByte(' ')
				}
			case extAst.AlignCenter:
				spaces := mr.columnWidths[column] - mr.stringWidth(string(cell))
				for i := 0; i < spaces/2; i++ {
					mr.buf.WriteByte(' ')
				}
				mr.buf.Write(cell)
				for i := 0; i < spaces-(spaces/2); i++ {
					mr.buf.WriteByte(' ')
				}
			case extAst.AlignRight:
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

// Span-level callbacks.

func (mr *MarkdownFmtRenderer) codeSpan(n *ast.CodeSpan, source []byte) {
	mr.buf.WriteByte('`')
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		segment := c.(*ast.Text).Segment
		value := segment.Value(source)
		if bytes.HasSuffix(value, []byte("\n")) {
			mr.buf.Write(value[:len(value)-1])
			if c != n.LastChild() {
				mr.buf.Write([]byte(" "))
			}
		} else {
			mr.buf.Write(value)
		}
	}
	mr.buf.WriteByte('`')
}

func (mr *MarkdownFmtRenderer) emphasis(node *ast.Emphasis, content []byte) {
	if len(content) == 0 {
		return
	}
	str := strings.Repeat("*", node.Level)
	mr.buf.WriteString(str)
	mr.buf.Write(content)
	mr.buf.WriteString(str)
}

func (mr *MarkdownFmtRenderer) image(link []byte, title []byte, alt []byte) {
	mr.buf.WriteString("![")
	mr.buf.Write(alt)
	mr.buf.WriteString("](")
	mr.buf.Write(link)
	if len(title) != 0 {
		mr.buf.WriteString(` "`)
		mr.buf.Write(title)
		mr.buf.WriteString(`"`)
	}
	mr.buf.WriteString(")")
}

func (mr *MarkdownFmtRenderer) lineBreak() {
	mr.buf.WriteString("  \n")

	spaces := mr.leader()
	mr.buf.Write(spaces)
}

func (mr *MarkdownFmtRenderer) link(link []byte, title []byte, content []byte) {
	mr.buf.WriteString("[")
	mr.buf.Write(content)
	mr.buf.WriteString("](")
	mr.buf.Write(link)
	if len(title) != 0 {
		mr.buf.WriteString(` "`)
		mr.buf.Write(title)
		mr.buf.WriteString(`"`)
	}
	mr.buf.WriteString(")")
}

func (mr *MarkdownFmtRenderer) strikeThrough(content []byte) {
	if len(content) == 0 {
		return
	}
	mr.buf.WriteString("~~")
	mr.buf.Write(content)
	mr.buf.WriteString("~~")
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

func (mr *MarkdownFmtRenderer) string(node *ast.String, source []byte, entering bool) {
	if !entering {
		return
	}
	// who knows
	mr.buf.Write(node.Value)
}

func (mr *MarkdownFmtRenderer) normalText(node *ast.Text, source []byte, entering bool) {
	if !entering {
		return
	}
	isHardLine := false
	text := node.Segment.Value(source)
	if node.HardLineBreak() {
		isHardLine = true
	} else if node.SoftLineBreak() {
		text = append(text, ' ')
	}
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
	if isHardLine {
		mr.lineBreak()
	}
}

func (mr *MarkdownFmtRenderer) skipSpaceIfNeededNormalText(cleanString string) bool {
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

func NewRenderer() *MarkdownFmtRenderer {
	return &MarkdownFmtRenderer{
		normalTextMarker:   make(map[*bytes.Buffer]int),
		orderedListCounter: make(map[int]int),
		listParagraph:      make(map[int]bool),

		buf: bytes.NewBuffer(nil),
	}
}

func NewParser() parser.Parser {
	return NewGoldmark().Parser()
}

func NewGoldmark() goldmark.Markdown {
	mr := NewRenderer()

	extensions := []goldmark.Extender{
		extension.Table,         // we need this to enable | tables |
		extension.Strikethrough, // we need this to enable ~~strike~~
	}
	parserOptions := []parser.Option{
		parser.WithAttribute(), // we need this to enable # headers {#custom-ids}
	}

	gm := goldmark.New(
		goldmark.WithExtensions(
			extensions...,
		),
		goldmark.WithParserOptions(
			parserOptions...,
		),
	)

	gm.SetRenderer(mr)
	return gm
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
	gm := NewGoldmark()
	output := bytes.NewBuffer(nil)
	err = gm.Convert(text, output)
	if err != nil {
		return nil, err
	} else {
		return output.Bytes(), nil
	}
}

// If src != nil, readSource returns src.
// If src == nil, readSource returns the result of reading the file specified by filename.
func readSource(filename string, src []byte) ([]byte, error) {
	if src != nil {
		return src, nil
	}
	return ioutil.ReadFile(filename)
}
