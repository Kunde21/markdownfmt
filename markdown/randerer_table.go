package markdown

import (
	"bytes"

	"github.com/mattn/go-runewidth"
	"github.com/pkg/errors"
	"github.com/yuin/goldmark/ast"
	extAST "github.com/yuin/goldmark/extension/ast"
)

func (r *render) renderTable(node *extAST.Table) error {
	var (
		columnAligns []extAST.Alignment
		columnWidths []int
		colIndex     int
		cellBuf      bytes.Buffer
	)

	// Walk tree initially to count column widths and alignments.
	for n := node.FirstChild(); n != nil; n = n.NextSibling() {
		if err := ast.Walk(n, func(inner ast.Node, entering bool) (ast.WalkStatus, error) {
			switch tnode := inner.(type) {
			case *extAST.TableRow, *extAST.TableHeader:
				if entering {
					colIndex = 0
				}
			case *extAST.TableCell:
				if entering {
					if _, isHeader := tnode.Parent().(*extAST.TableHeader); isHeader {
						columnAligns = append(columnAligns, tnode.Alignment)
					}

					cellBuf.Reset()
					if err := ast.Walk(tnode, r.mr.newRender(&cellBuf, r.source).renderNode); err != nil {
						return ast.WalkStop, err
					}
					width := runewidth.StringWidth(cellBuf.String())
					if len(columnWidths) <= colIndex {
						columnWidths = append(columnWidths, width)
					} else if width > columnWidths[colIndex] {
						columnWidths[colIndex] = width
					}
					colIndex++
					return ast.WalkSkipChildren, nil
				}
			default:
				return ast.WalkStop, errors.Errorf("detected unexpected tree type %s", tnode.Kind().String())
			}
			return ast.WalkContinue, nil
		}); err != nil {
			return err
		}
	}

	// Write all according to alignments and width.
	for n := node.FirstChild(); n != nil; n = n.NextSibling() {
		if err := ast.Walk(n, func(inner ast.Node, entering bool) (ast.WalkStatus, error) {
			switch tnode := inner.(type) {
			case *extAST.TableRow:
				if entering {
					colIndex = 0
					_, _ = r.w.Write(newLineChar)
				} else {
					_, _ = r.w.Write([]byte("|"))
				}
			case *extAST.TableHeader:
				if entering {
					colIndex = 0
				} else {
					_, _ = r.w.Write([]byte("|\n"))
					for i, align := range columnAligns {
						width := columnWidths[i]
						_, _ = r.w.Write([]byte{'|'})
						if align == extAST.AlignLeft ||
							align == extAST.AlignCenter {
							_, _ = r.w.Write([]byte{':'})
						} else {
							_, _ = r.w.Write([]byte{'-'})
						}
						for ; width > 0; width-- {
							_, _ = r.w.Write([]byte{'-'})
						}
						if align == extAST.AlignRight ||
							align == extAST.AlignCenter {
							_, _ = r.w.Write([]byte{':'})
						} else {
							_, _ = r.w.Write([]byte{'-'})
						}
					}
					_, _ = r.w.Write([]byte("|"))
				}
			case *extAST.TableCell:
				if entering {
					width := columnWidths[colIndex]
					align := columnAligns[colIndex]

					if tnode.Parent().Kind() == extAST.KindTableHeader {
						align = extAST.AlignLeft
					}

					cellBuf.Reset()
					if err := ast.Walk(tnode, r.mr.newRender(&cellBuf, r.source).renderNode); err != nil {
						return ast.WalkStop, err
					}

					_, _ = r.w.Write([]byte("| "))
					whitespaceWidth := width - runewidth.StringWidth(cellBuf.String())
					switch align {
					default:
						fallthrough
					case extAST.AlignLeft:
						_, _ = r.w.Write(cellBuf.Bytes())
						_, _ = r.w.Write(bytes.Repeat([]byte{' '}, 1+whitespaceWidth))
					case extAST.AlignCenter:
						first := whitespaceWidth / 2
						_, _ = r.w.Write(bytes.Repeat([]byte{' '}, first))
						_, _ = r.w.Write(cellBuf.Bytes())
						_, _ = r.w.Write(bytes.Repeat([]byte{' '}, whitespaceWidth-first))
						_, _ = r.w.Write([]byte{' '})
					case extAST.AlignRight:
						_, _ = r.w.Write(bytes.Repeat([]byte{' '}, whitespaceWidth))
						_, _ = r.w.Write(cellBuf.Bytes())
						_, _ = r.w.Write([]byte{' '})
					}
					colIndex++
					return ast.WalkSkipChildren, nil
				}
			default:
				return ast.WalkStop, errors.Errorf("detected unexpected tree type %s", tnode.Kind().String())
			}
			return ast.WalkContinue, nil
		}); err != nil {
			return err
		}
	}
	return nil
}
