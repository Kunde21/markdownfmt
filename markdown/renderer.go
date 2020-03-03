// Package renderer renders the given AST to certain formats.
package markdown

import (
	"bytes"
	"io"

	"github.com/yuin/goldmark/ast"
	extAst "github.com/yuin/goldmark/extension/ast"

	"github.com/yuin/goldmark/renderer"
)

func (mr *MarkdownFmtRenderer) RenderSingle(writer *bytes.Buffer, source []byte, n ast.Node, entering bool) ast.WalkStatus {

	switch tnode := n.(type) {
	case *ast.Document:
		break
	case *ast.TextBlock:
		mr.paragraph(tnode, entering)
	case *ast.Paragraph:
		mr.paragraph(tnode, entering)
	case *ast.Heading:
		if entering {
			children := mr.renderChildren(source, n)
			mr.header(tnode, children)
		}
		return ast.WalkSkipChildren
	case *ast.Text:
		mr.normalText(tnode, source, entering)
	case *ast.String:
		mr.string(tnode, source, entering)
	case *ast.CodeSpan:
		if entering {
			mr.codeSpan(tnode, source)
		}
		return ast.WalkSkipChildren
	case *extAst.Strikethrough:
		if entering {
			children := mr.renderChildren(source, n)
			mr.strikeThrough(children)
		}
		return ast.WalkSkipChildren
	case *ast.Emphasis:
		if entering {
			children := mr.renderChildren(source, n)
			mr.emphasis(tnode, children)
		}
		return ast.WalkSkipChildren
	case *ast.ThematicBreak:
		if entering {
			mr.hrule()
		}
	case *ast.Blockquote:
		mr.blockQuote(entering)
	case *ast.List:
		mr.list(tnode, entering)
	case *ast.ListItem:
		mr.item(tnode, entering, source)
	case *ast.Link:
		if entering {
			children := mr.renderChildren(source, n)
			mr.link(tnode.Destination, tnode.Title, children)
		}
		return ast.WalkSkipChildren
	case *ast.Image:
		if entering {
			children := mr.renderChildren(source, n)
			mr.image(tnode.Destination, tnode.Title, children)
		}
		return ast.WalkSkipChildren
	case *ast.CodeBlock:
		if entering {
			mr.blockCode(tnode, source)
		}
	case *ast.FencedCodeBlock:
		if entering {
			mr.blockCode(tnode, source)
		}
	case *ast.HTMLBlock:
		if entering {
			mr.blockHtml(tnode, source)
		}
	case *ast.RawHTML:
		if entering {
			mr.rawHtml(tnode, source)
		}
		return ast.WalkSkipChildren
	case *extAst.Table:
		mr.table(tnode, entering)
	case *extAst.TableHeader:
		if entering {
			mr.tableIsHeader = true
		}
	case *extAst.TableRow:
		if entering {
			mr.tableIsHeader = false
		}
	case *extAst.TableCell:
		if entering {
			children := mr.renderChildren(source, n)
			if mr.tableIsHeader {
				mr.tableHeaderCell(children, tnode.Alignment)
			} else {
				mr.tableCell(children)
			}
		}
		return ast.WalkSkipChildren
	default:
		panic("unknown type" + n.Kind().String())
	}

	if !entering {
		mr.buf.WriteTo(writer)
		mr.buf.Reset()
		mr.buf = bytes.NewBuffer(nil)
	}

	return ast.WalkContinue
}

func (mr *MarkdownFmtRenderer) renderChildren(source []byte, node ast.Node) []byte {
	oldBuf := mr.buf
	mr.buf = bytes.NewBuffer(nil)
	mr.normalTextMarker = map[*bytes.Buffer]int{}
	resBuf := bytes.NewBuffer(nil)
	for n := node.FirstChild(); n != nil; n = n.NextSibling() {
		ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			return mr.RenderSingle(resBuf, source, n, entering), nil
		})
	}
	resBuf.Write(mr.buf.Bytes())
	mr.buf = oldBuf
	return resBuf.Bytes()
}

// Render renders the given AST node to the given writer with the given Renderer.
func (mr *MarkdownFmtRenderer) Render(w io.Writer, source []byte, n ast.Node) error {
	resBuf := mr.renderChildren(source, n)
	resBuf = bytes.TrimLeft(resBuf, "\n")
	_, err := w.Write(resBuf)
	return err
}

func (mr *MarkdownFmtRenderer) AddOptions(...renderer.Option) {
	panic("aa")
}
