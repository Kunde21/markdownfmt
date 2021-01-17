// Package renderer renders the given AST to certain formats.
package markdown

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/yuin/goldmark/ast"
	extAST "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
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

func NewRenderer() *Renderer {
	return &Renderer{}
}

// render represents a single markdown rendering operation.
type render struct {
	mr *Renderer

	// TODO(bwplotka): Wrap it with something that catch errors.
	w      *lineIndentWriter
	source []byte
}

func (mr *Renderer) newRender(w io.Writer, source []byte) *render {
	return &render{
		mr:     mr,
		w:      wrapWithLineIndentWriter(w),
		source: source,
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
			*ast.Blockquote, *ast.HTMLBlock:
			_, _ = r.w.Write(newLineChar)
			_, _ = r.w.Write(newLineChar)
		case *ast.List:
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
			_, _ = r.w.Write(spaceChar)
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
		return r.wrapNonEmptyContentWith(bytes.Repeat([]byte{'*'}, tnode.Level), entering), nil
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

		l := tnode.Segments.Len()
		for i := 0; i < l; i++ {
			segment := tnode.Segments.At(i)
			_, _ = r.w.Write(segment.Value(r.source))
		}

	// Blocks.
	case *ast.Paragraph:
		break
	case *ast.TextBlock:
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

		_, _ = r.w.Write(newLineChar)
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

		switch noAllocString(lang) {
		case "Go", "go":
			gofmt, err := format.Source(codeBuf.Bytes())
			if err != nil {
				// We don't handle gofmt errors. If code is not compilable we just don't format it without any warning.
				_, _ = r.w.Write(codeBuf.Bytes())
				break
			}
			_, _ = r.w.Write(gofmt)
		default:
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
	case *ast.List:
		break
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
	case *extAST.TableCell:
		break
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
