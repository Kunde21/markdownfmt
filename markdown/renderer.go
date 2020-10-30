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
	newLineChar        = []byte{'\n'}
	spaceChar          = []byte{' '}
	strikeThroughChars = []byte("~~")
	thematicBreakChars = []byte("---")
	blockquoteChars    = []byte{'>', ' '}
	codeBlockChars     = []byte("```")
)

// Ensure compatibility with Goldmark parser.
var _ renderer.Renderer = &Renderer{}

// Renderer allows to render markdown AST into markdown bytes in consistent format.
// Render is reusable across Renders, it holds configuration only.
type Renderer struct{}

func (mr *Renderer) AddOptions(opts ...renderer.Option) {
	c := renderer.Config{}
	for _, o := range opts {
		o.SetConfig(&c)
	}

	// TODO(bwplotka): Add headers optionality (https://github.com/Kunde21/markdownfmt/issues/14).
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
		// All Block types (except few) usually have 2x new lines if not first in siblings.
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
			if node.HasBlankPreviousLines() {
				_, _ = r.w.Write(newLineChar)
			}
		}
	}

	switch tnode := node.(type) {
	case *ast.Document:
		if !entering {
			_, _ = r.w.Write(newLineChar)
		}

	// Spans, meaning no newlines before or after.
	case *ast.Text:
		if entering {

			text := tnode.Segment.Value(r.source)
			clean := cleanWithoutTrim(text)
			if len(clean) == 0 {
				// Nothing to render.
				return ast.WalkContinue, nil
			}
			_, _ = r.w.Write(clean)
			return ast.WalkContinue, nil
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
		_, _ = r.w.Write(tnode.Value)
	case *ast.CodeSpan:
		if entering {
			_, _ = r.w.Write([]byte{'`'})
			return ast.WalkContinue, nil
		}
		_, _ = r.w.Write([]byte{'`'})
	case *extAST.Strikethrough:
		return r.wrapNonEmptyContentWith(strikeThroughChars, entering), nil
	case *ast.Emphasis:
		return r.wrapNonEmptyContentWith(bytes.Repeat([]byte{'*'}, tnode.Level), entering), nil
	case *ast.Link:
		if entering {
			r.w.AddIndentOnFirstWrite([]byte("["))
			return ast.WalkContinue, nil
		}
		_, _ = fmt.Fprintf(r.w, "](%s", tnode.Destination)
		if len(tnode.Title) > 0 {
			_, _ = fmt.Fprintf(r.w, ` "%s"`, tnode.Title)
		}
		_, _ = r.w.Write([]byte{')'})
	case *ast.Image:
		if entering {
			r.w.AddIndentOnFirstWrite([]byte("!["))
			return ast.WalkContinue, nil
		}
		_, _ = fmt.Fprintf(r.w, "](%s", tnode.Destination)
		if len(tnode.Title) > 0 {
			_, _ = fmt.Fprintf(r.w, ` "%s"`, tnode.Title)
		}
		_, _ = r.w.Write([]byte{')'})
	case *ast.RawHTML:
		if entering {
			l := tnode.Segments.Len()
			for i := 0; i < l; i++ {
				segment := tnode.Segments.At(i)
				_, _ = r.w.Write(segment.Value(r.source))
			}
		}

	// Blocks.
	case *ast.Paragraph:
		break
	case *ast.TextBlock:
		break
	case *ast.Heading:
		if entering {
			_, _ = r.w.Write(bytes.Repeat([]byte{'#'}, tnode.Level))
			_, _ = r.w.Write(spaceChar)
			return ast.WalkContinue, nil
		}

		id, hasId := node.AttributeString("id")
		if hasId {
			_, _ = fmt.Fprintf(r.w, " {#%s}", id)
		}
	case *ast.HTMLBlock:
		if entering {
			_, _ = r.w.Write(newLineChar)
			return ast.WalkContinue, nil
		}
	case *ast.CodeBlock, *ast.FencedCodeBlock:
		if entering {
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
					return ast.WalkStop, err
				}
				_, _ = r.w.Write(gofmt)

			default:
				_, _ = r.w.Write(codeBuf.Bytes())
			}

			_, _ = r.w.Write(codeBlockChars)
			return ast.WalkSkipChildren, nil
		}
	case *ast.ThematicBreak:
		if entering {
			_, _ = r.w.Write(thematicBreakChars)
		}
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
		} else {
			if tnode.NextSibling() != nil && tnode.NextSibling().Kind() == ast.KindListItem {
				_, _ = r.w.Write(newLineChar)
			}
		}
		r.w.UpdateIndent(tnode, entering)

	case *extAST.Table:
		if entering {
			// Render it straight away. No nested tables are supported and we expect
			// tables to have limited content, so limit WALK.
			if err := r.renderTable(tnode); err != nil {
				return ast.WalkStop, errors.Wrap(err, "rendering table")
			}
			return ast.WalkSkipChildren, nil
		}
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

func noAllocString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}

func isNumber(data []byte) bool {
	for _, b := range data {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
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
