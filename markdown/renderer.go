package markdown

import (
	"bytes"
	"io"
	"log"

	blackfriday "gopkg.in/russross/blackfriday.v2"
)

func (mr *markdownRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	switch node.Type {
	case blackfriday.Document:
		if !entering {
			mr.doubleSpace(nil)
		}
		break
	case blackfriday.BlockQuote:
	case blackfriday.List:
		mr.List(mr.buf, node, entering)
	case blackfriday.Item:
		mr.ListItem(mr.buf, node, entering)
	case blackfriday.Paragraph:
		mr.Paragraph(mr.buf, entering)
	case blackfriday.Heading:
		mr.Header(mr.buf, node, entering)
	case blackfriday.HorizontalRule:
	case blackfriday.Emph:
	case blackfriday.Strong:
	case blackfriday.Del:
	case blackfriday.Link:
	case blackfriday.Image:
	case blackfriday.Text:
		mr.NormalText(mr.buf, node)
	case blackfriday.HTMLBlock:
	case blackfriday.CodeBlock:
		mr.BlockCode(mr.buf, node, string(node.Info))
	case blackfriday.Code:
		mr.CodeSpan(mr.buf, node.Literal)
	case blackfriday.Softbreak:
	case blackfriday.Hardbreak:
	case blackfriday.HTMLSpan:
	case blackfriday.Table:
		mr.Table(mr.buf, node, entering)
	case blackfriday.TableHead:
	case blackfriday.TableBody:
	case blackfriday.TableRow:
	case blackfriday.TableCell:
		buf := mr.buf
		mr.buf = bytes.NewBuffer(nil)
		for n := node.FirstChild; n != nil; n = n.Next {
			n.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
				return mr.RenderNode(mr.buf, n, entering)
			})
		}
		if node.IsHeader {
			mr.TableHeaderCell(mr.buf, mr.buf.Bytes(), node.Align)
		} else {
			mr.TableCell(mr.buf, mr.buf.Bytes(), node.Align)
		}
		mr.buf = buf
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
