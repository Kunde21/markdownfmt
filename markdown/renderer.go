package markdown

import (
	"bytes"
	"io"
	"log"

	"github.com/russross/blackfriday/v2"
)

func (mr *markdownRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	switch node.Type {
	case blackfriday.Document:
		break
	case blackfriday.BlockQuote:
		mr.BlockQuote(node, entering)
	case blackfriday.List:
		mr.List(node, entering)
	case blackfriday.Item:
		mr.item(node, entering)
	case blackfriday.Paragraph:
		mr.paragraph(node, entering)
	case blackfriday.Heading:
		children := mr.renderChildren(node)
		mr.Header(node, children)
		return blackfriday.SkipChildren
	case blackfriday.HorizontalRule:
		mr.HRule()
	case blackfriday.Emph:
		children := mr.renderChildren(node)
		mr.emphasis(children)
		return blackfriday.SkipChildren
	case blackfriday.Strong:
		children := mr.renderChildren(node)
		mr.doubleEmphasis(children)
		return blackfriday.SkipChildren
	case blackfriday.Del:
		children := mr.renderChildren(node)
		mr.strikeThrough(children)
		return blackfriday.SkipChildren
	case blackfriday.Link:
		children := mr.renderChildren(node)
		mr.link(node.Destination, node.Title, children)
		return blackfriday.SkipChildren
	case blackfriday.Image:
		children := mr.renderChildren(node)
		mr.image(node.Destination, node.Title, children)
		return blackfriday.SkipChildren
	case blackfriday.Text:
		mr.NormalText(node)
	case blackfriday.HTMLBlock:
		mr.BlockHtml(node)
	case blackfriday.CodeBlock:
		mr.BlockCode(node, string(node.Info))
	case blackfriday.Code:
		mr.codeSpan(node.Literal)
	case blackfriday.Softbreak:
	case blackfriday.Hardbreak:
		mr.lineBreak()
	case blackfriday.HTMLSpan:
		mr.rawHtmlTag(node)
	case blackfriday.Table:
		mr.table(node, entering)
	case blackfriday.TableHead:
	case blackfriday.TableBody:
	case blackfriday.TableRow:
	case blackfriday.TableCell:
		children := mr.renderChildren(node)
		if node.IsHeader {
			mr.tableHeaderCell(children, node.Align)
		} else {
			mr.tableCell(children, node.Align)
		}
		return blackfriday.SkipChildren
	default:
		panic("unknown node type")
	}
	if !entering {
		_, err := mr.buf.WriteTo(w)
		if err != nil {
			log.Println(err)
		}
		mr.buf.Reset()
	}
	return blackfriday.GoToNext
}

func (mr *markdownRenderer) renderChildren(node *blackfriday.Node) []byte {
	oldBuf := mr.buf
	mr.buf = bytes.NewBuffer(nil)
	resBuf := bytes.NewBuffer(nil)
	for n := node.FirstChild; n != nil; n = n.Next {
		n.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
			return mr.RenderNode(resBuf, node, entering)
		})
	}
	mr.buf.WriteTo(resBuf)
	mr.buf = oldBuf
	return resBuf.Bytes()
}
