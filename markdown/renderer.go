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
		mr.BlockQuote(mr.buf, node, entering)
	case blackfriday.List:
		mr.List(mr.buf, node, entering)
	case blackfriday.Item:
		mr.item(mr.buf, node, entering)
	case blackfriday.Paragraph:
		mr.paragraph(mr.buf, entering)
	case blackfriday.Heading:
		mr.Header(node, entering)
	case blackfriday.HorizontalRule:
		mr.HRule()
	case blackfriday.Emph:
	case blackfriday.Strong:
	case blackfriday.Del:
	case blackfriday.Link:
		children := mr.renderChildren(node)
		mr.link(mr.buf, node.Destination, node.Title, children)
		return blackfriday.SkipChildren
	case blackfriday.Image:
	case blackfriday.Text:
		mr.NormalText(mr.buf, node)
	case blackfriday.HTMLBlock:
		mr.BlockHtml(node)
	case blackfriday.CodeBlock:
		mr.BlockCode(mr.buf, node, string(node.Info))
	case blackfriday.Code:
		mr.codeSpan(mr.buf, node.Literal)
	case blackfriday.Softbreak:
	case blackfriday.Hardbreak:
		mr.lineBreak(mr.buf)
	case blackfriday.HTMLSpan:
		mr.rawHtmlTag(node)
	case blackfriday.Table:
		mr.table(mr.buf, node, entering)
	case blackfriday.TableHead:
	case blackfriday.TableBody:
	case blackfriday.TableRow:
	case blackfriday.TableCell:
		children := mr.renderChildren(node)
		if node.IsHeader {
			mr.tableHeaderCell(mr.buf, children, node.Align)
		} else {
			mr.tableCell(mr.buf, children, node.Align)
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
	buf := mr.buf
	mr.buf = bytes.NewBuffer(nil)
	for n := node.FirstChild; n != nil; n = n.Next {
		n.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
			return mr.RenderNode(mr.buf, n, entering)
		})
	}
	buf, mr.buf = mr.buf, buf
	return buf.Bytes()
}
