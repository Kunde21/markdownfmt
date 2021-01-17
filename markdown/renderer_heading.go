package markdown

import (
	"bytes"
	"fmt"

	"github.com/mattn/go-runewidth"
	"github.com/yuin/goldmark/ast"
)

func (r *render) renderHeading(node *ast.Heading) error {
	underlineHeading := false
	if r.mr.underlineHeadings {
		underlineHeading = node.Level <= 2
	}

	if !underlineHeading {
		r.w.Write(bytes.Repeat([]byte{'#'}, node.Level))
		r.w.Write(spaceChar)
	}

	var headBuf bytes.Buffer
	headBuf.Reset()

	for n := node.FirstChild(); n != nil; n = n.NextSibling() {
		if err := ast.Walk(n, func(inner ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				if err := ast.Walk(inner, r.mr.newRender(&headBuf, r.source).renderNode); err != nil {
					return ast.WalkStop, err
				}
			}
			return ast.WalkSkipChildren, nil
		}); err != nil {
			return err
		}
	}

	id, hasId := node.AttributeString("id")
	if hasId {
		_, _ = fmt.Fprintf(&headBuf, " {#%s}", id)
	}

	_, _ = r.w.Write(headBuf.Bytes())

	if underlineHeading {
		width := runewidth.StringWidth(headBuf.String())

		_, _ = r.w.Write(newLineChar)

		switch node.Level {
		case 1:
			r.w.Write(bytes.Repeat(heading1UnderlineChar, width))
		case 2:
			r.w.Write(bytes.Repeat(heading2UnderlineChar, width))
		}
	}

	return nil
}
