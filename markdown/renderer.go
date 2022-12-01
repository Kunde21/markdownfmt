package markdown

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"unicode/utf8"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/yuin/goldmark/ast"
	extAST "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
)

var (
	newLineChar             = []byte{'\n'}
	spaceChar               = []byte{' '}
	strikeThroughChars      = []byte("~~")
	thematicBreakChars      = []byte("---")
	blockquoteChars         = []byte{'>', ' '}
	codeBlockChars          = []byte("```")
	tableHeaderColChar      = []byte{'-'}
	tableHeaderAlignColChar = []byte{':'}
	heading1UnderlineChar   = []byte{'='}
	heading2UnderlineChar   = []byte{'-'}
)

// Ensure compatibility with Goldmark parser.
var _ renderer.Renderer = &Renderer{}

// Renderer allows to render markdown AST into markdown bytes in consistent format.
// Render is reusable across Renders, it holds configuration only.
type Renderer struct {
	underlineHeadings bool
	softWraps         bool
	emphToken         []byte
	strongToken       []byte // if nil, use emphToken*2

	// language name => format function
	formatters map[string]func([]byte) []byte
}

func (mr *Renderer) AddOptions(...renderer.Option) {
	// goldmark weirdness, just ignore (called with just HTML options...)
}

func (mr *Renderer) AddMarkdownOptions(opts ...Option) {
	for _, o := range opts {
		o(mr)
	}
}

type Option func(r *Renderer)

func WithUnderlineHeadings() Option {
	return func(r *Renderer) {
		r.underlineHeadings = true
	}
}

// WithSoftWraps allows you to wrap lines even on soft line breaks.
func WithSoftWraps() Option {
	return func(r *Renderer) {
		r.softWraps = true
	}
}

// WithEmphasisToken specifies the character used to wrap emphasised text.
// Per the CommonMark spec, valid values are '*' and '_'.
//
// Defaults to '*'.
func WithEmphasisToken(c rune) Option {
	return func(r *Renderer) {
		r.emphToken = utf8.AppendRune(nil, c)
	}
}

// WithStrongToken specifies the string used to wrap bold text.
// Per the CommonMark spec, valid values are '**' and '__'.
//
// Defaults to repeating the emphasis token twice.
// See [WithEmphasisToken] for how to change that.
func WithStrongToken(s string) Option {
	return func(r *Renderer) {
		r.strongToken = []byte(s)
	}
}

// CodeFormatter reformats code samples found in the document,
// matching them by name.
type CodeFormatter struct {
	// Name of the language.
	Name string

	// Aliases for the language, if any.
	Aliases []string

	// Function to format the code snippet.
	// In case of errors, format functions should typically return
	// the original string unchanged.
	Format func([]byte) []byte
}

// WithCodeFormatters changes the functions used to reformat code blocks found
// in the original file.
//
//	formatters := DefaultCodeFormatters()
//	formatters = append(formatters, ...)
//
//	r := NewRenderer()
//	r.AddMarkdownOptions(WithCodeFormatters(formatters))
//
// Pass an empty list to disable code formatting.
//
//	r := NewRenderer()
//	r.AddMarkdownOptions(WithCodeFormatters())
//
// Defaults to DefaultCodeFormatters.
func WithCodeFormatters(fs ...CodeFormatter) Option {
	return func(r *Renderer) {
		formatters := make(map[string]func([]byte) []byte, len(fs))
		for _, f := range fs {
			formatters[f.Name] = f.Format
			for _, alias := range f.Aliases {
				formatters[alias] = f.Format
			}
		}
		r.formatters = formatters
	}
}

// DefaultCodeFormatters reports the list of default code formatters
// used by the system.
//
// Replace this with WithCodeFormatters.
func DefaultCodeFormatters() []CodeFormatter {
	return []CodeFormatter{
		{
			Name:    "go",
			Aliases: []string{"Go"},
			Format: func(src []byte) []byte {
				gofmt, err := format.Source(src)
				if err != nil {
					// We don't handle gofmt errors.
					// If code is not compilable we just
					// don't format it without any warning.
					return src
				}
				return gofmt
			},
		},
	}
}

func NewRenderer() *Renderer {
	r := &Renderer{
		emphToken: []byte{'*'},
		// Leave strongToken as nil by default.
		// At render time, we'll use what was specified,
		// or repeat emphToken twice to get the strong token.
	}
	r.AddMarkdownOptions(WithCodeFormatters(DefaultCodeFormatters()...))
	return r
}

// render represents a single markdown rendering operation.
type render struct {
	mr *Renderer

	emphToken   []byte
	strongToken []byte

	// TODO(bwplotka): Wrap it with something that catch errors.
	w      *lineIndentWriter
	source []byte
}

func (mr *Renderer) newRender(w io.Writer, source []byte) *render {
	strongToken := mr.strongToken
	if len(strongToken) == 0 {
		strongToken = bytes.Repeat(mr.emphToken, 2)
	}

	return &render{
		mr:          mr,
		w:           wrapWithLineIndentWriter(w),
		source:      source,
		strongToken: strongToken,
		emphToken:   mr.emphToken,
	}
}

// Render renders the given AST node to the given buffer with the given Renderer.
// NOTE: This is the entry point used by Goldmark.
func (mr *Renderer) Render(w io.Writer, source []byte, node ast.Node) error {
	// Perform DFS.
	return ast.Walk(node, mr.newRender(w, source).renderNode)
}

func (r *render) renderNode(node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering && node.PreviousSibling() != nil {
		switch node.(type) {
		// All Block types (except few) usually have 2x new lines before itself when they are non-first siblings.
		case *ast.Paragraph, *ast.Heading, *ast.FencedCodeBlock,
			*ast.CodeBlock, *ast.ThematicBreak, *extAST.Table,
			*ast.Blockquote:
			_, _ = r.w.Write(newLineChar)
			_, _ = r.w.Write(newLineChar)
		case *ast.List, *ast.HTMLBlock:
			_, _ = r.w.Write(newLineChar)
			if node.HasBlankPreviousLines() {
				_, _ = r.w.Write(newLineChar)
			}
		case *ast.ListItem:
			// TODO(bwplotka): Handle tight/loose rule explicitly.
			// See: https://github.github.com/gfm/#loose
			if node.HasBlankPreviousLines() {
				_, _ = r.w.Write(newLineChar)
			}
		}
	}

	switch tnode := node.(type) {
	case *ast.Document:
		if entering {
			break
		}

		_, _ = r.w.Write(newLineChar)

	// Spans, meaning no newlines before or after.
	case *ast.Text:
		if entering {
			text := tnode.Segment.Value(r.source)
			clean := cleanWithoutTrim(text)
			if len(clean) == 0 {
				// Nothing to render.
				break
			}
			_, _ = r.w.Write(clean)
			break
		}

		if tnode.SoftLineBreak() {
			char := spaceChar
			if r.mr.softWraps {
				char = newLineChar
			}
			_, _ = r.w.Write(char)
		}

		if tnode.HardLineBreak() {
			if tnode.SoftLineBreak() {
				_, _ = r.w.Write(spaceChar)
			}
			_, _ = r.w.Write(newLineChar)
		}
	case *ast.String:
		if entering {
			_, _ = r.w.Write(tnode.Value)
		}
	case *ast.AutoLink:
		// We treat autolink as normal string.
		if entering {
			_, _ = r.w.Write(tnode.Label(r.source))
		}
	case *extAST.TaskCheckBox:
		if !entering {
			break
		}
		if tnode.IsChecked {
			_, _ = r.w.Write([]byte("[X] "))
			break
		}
		_, _ = r.w.Write([]byte("[ ] "))
	case *ast.CodeSpan:
		if entering {
			_, _ = r.w.Write([]byte{'`'})
			break
		}

		_, _ = r.w.Write([]byte{'`'})
	case *extAST.Strikethrough:
		return r.wrapNonEmptyContentWith(strikeThroughChars, entering), nil
	case *ast.Emphasis:
		var emWrapper []byte
		switch tnode.Level {
		case 1:
			emWrapper = r.emphToken
		case 2:
			emWrapper = r.strongToken
		default:
			emWrapper = bytes.Repeat(r.emphToken, tnode.Level)
		}
		return r.wrapNonEmptyContentWith(emWrapper, entering), nil
	case *ast.Link:
		if entering {
			r.w.AddIndentOnFirstWrite([]byte("["))
			break
		}

		_, _ = fmt.Fprintf(r.w, "](%s", tnode.Destination)
		if len(tnode.Title) > 0 {
			_, _ = fmt.Fprintf(r.w, ` "%s"`, tnode.Title)
		}
		_, _ = r.w.Write([]byte{')'})
	case *ast.Image:
		if entering {
			r.w.AddIndentOnFirstWrite([]byte("!["))
			break
		}

		_, _ = fmt.Fprintf(r.w, "](%s", tnode.Destination)
		if len(tnode.Title) > 0 {
			_, _ = fmt.Fprintf(r.w, ` "%s"`, tnode.Title)
		}
		_, _ = r.w.Write([]byte{')'})
	case *ast.RawHTML:
		if !entering {
			break
		}

		for i := 0; i < tnode.Segments.Len(); i++ {
			segment := tnode.Segments.At(i)
			_, _ = r.w.Write(segment.Value(r.source))
		}
		return ast.WalkSkipChildren, nil

	// Blocks.
	case *ast.Paragraph, *ast.TextBlock, *ast.List, *extAST.TableCell:
		// Things that has no content, just children elements, go there.
		break
	case *ast.Heading:
		if !entering {
			break
		}

		// Render it straight away. No nested headings are supported and we expect
		// headings to have limited content, so limit WALK.
		if err := r.renderHeading(tnode); err != nil {
			return ast.WalkStop, errors.Wrap(err, "rendering heading")
		}
		return ast.WalkSkipChildren, nil
	case *ast.HTMLBlock:
		if !entering {
			break
		}

		var segments []text.Segment
		for i := 0; i < node.Lines().Len(); i++ {
			segments = append(segments, node.Lines().At(i))
		}

		if tnode.ClosureLine.Len() != 0 {
			segments = append(segments, tnode.ClosureLine)
		}
		for i, s := range segments {
			o := s.Value(r.source)
			if i == len(segments)-1 {
				o = bytes.TrimSuffix(o, []byte("\n"))
			}
			_, _ = r.w.Write(o)
		}
		return ast.WalkSkipChildren, nil
	case *ast.CodeBlock, *ast.FencedCodeBlock:
		if !entering {
			break
		}

		_, _ = r.w.Write(codeBlockChars)

		var lang []byte
		if fencedNode, isFenced := node.(*ast.FencedCodeBlock); isFenced && fencedNode.Info != nil {
			lang = fencedNode.Info.Text(r.source)
			_, _ = r.w.Write(lang)
			for _, elt := range bytes.Fields(lang) {
				elt = bytes.TrimSpace(bytes.TrimLeft(elt, ". "))
				if len(elt) == 0 {
					continue
				}
				lang = elt
				break
			}
		}

		_, _ = r.w.Write(newLineChar)
		codeBuf := bytes.Buffer{}
		for i := 0; i < tnode.Lines().Len(); i++ {
			line := tnode.Lines().At(i)
			_, _ = codeBuf.Write(line.Value(r.source))
		}

		if formatCode, ok := r.mr.formatters[noAllocString(lang)]; ok {
			code := formatCode(codeBuf.Bytes())
			if !bytes.HasSuffix(code, newLineChar) {
				// Ensure code sample ends with a newline.
				code = append(code, newLineChar...)
			}
			_, _ = r.w.Write(code)
		} else {
			_, _ = r.w.Write(codeBuf.Bytes())
		}

		_, _ = r.w.Write(codeBlockChars)
		return ast.WalkSkipChildren, nil
	case *ast.ThematicBreak:
		if !entering {
			break
		}

		_, _ = r.w.Write(thematicBreakChars)
	case *ast.Blockquote:
		r.w.UpdateIndent(tnode, entering)

		if entering && node.Parent() != nil && node.Parent().Kind() == ast.KindListItem &&
			node.PreviousSibling() == nil {
			_, _ = r.w.Write(blockquoteChars)
		}

	case *ast.ListItem:
		if entering {
			_, _ = r.w.Write(listItemMarkerChars(tnode))
		} else if tnode.NextSibling() != nil && tnode.NextSibling().Kind() == ast.KindListItem {
			// Newline after list item.
			_, _ = r.w.Write(newLineChar)
		}
		r.w.UpdateIndent(tnode, entering)

	case *extAST.Table:
		if !entering {
			break
		}

		// Render it straight away. No nested tables are supported and we expect
		// tables to have limited content, so limit WALK.
		if err := r.renderTable(tnode); err != nil {
			return ast.WalkStop, errors.Wrap(err, "rendering table")
		}
		return ast.WalkSkipChildren, nil
	case *extAST.TableRow, *extAST.TableHeader:
		return ast.WalkStop, errors.Errorf("%v element detected, but table should be rendered in renderTable instead", tnode.Kind().String())
	default:
		return ast.WalkStop, errors.Errorf("detected unexpected tree type %s", tnode.Kind().String())
	}
	return ast.WalkContinue, nil
}

func (r *render) wrapNonEmptyContentWith(b []byte, entering bool) ast.WalkStatus {
	if entering {
		r.w.AddIndentOnFirstWrite(b)
		return ast.WalkContinue
	}

	if r.w.WasIndentOnFirstWriteWritten() {
		_, _ = r.w.Write(b)
		return ast.WalkContinue
	}
	r.w.DelIndentOnFirstWrite(b)
	return ast.WalkContinue
}

func listItemMarkerChars(tnode *ast.ListItem) []byte {
	parList := tnode.Parent().(*ast.List)
	if parList.IsOrdered() {
		cnt := 1
		if parList.Start != 0 {
			cnt = parList.Start
		}
		s := tnode.PreviousSibling()
		for s != nil {
			cnt++
			s = s.PreviousSibling()
		}
		return []byte(fmt.Sprintf("%d%c ", cnt, parList.Marker))
	}
	return []byte{parList.Marker, spaceChar[0]}
}

func noAllocString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}

// cleanWithoutTrim is like clean, but doesn't trim blanks.
func cleanWithoutTrim(b []byte) []byte {
	var ret []byte
	var p byte
	for i := 0; i < len(b); i++ {
		q := b[i]
		if q == '\n' || q == '\r' || q == '\t' {
			q = ' '
		}
		if q != ' ' || p != ' ' {
			ret = append(ret, q)
			p = q
		}
	}
	return ret
}
