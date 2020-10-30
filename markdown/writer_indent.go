package markdown

import (
	"bytes"
	"io"

	"github.com/yuin/goldmark/ast"
)

// lineIndentWriter wraps io.Writer and adds given indent everytime new line is created .
type lineIndentWriter struct {
	io.Writer

	indent                []byte
	whitespace            []byte
	firstWriteExtraIndent []byte

	previousCharWasNewLine bool
}

func wrapWithLineIndentWriter(w io.Writer) *lineIndentWriter {
	return &lineIndentWriter{Writer: w, previousCharWasNewLine: true}
}

func (l *lineIndentWriter) UpdateIndent(node ast.Node, entering bool) {
	// Recalculate indent.
	l.indent = l.indent[:0]

	p := node
	if !entering {
		p = p.Parent()
	}

	for ; p != nil; p = p.Parent() {
		if p.Kind() == ast.KindBlockquote {
			// Prepend, as we go from down.
			l.indent = append(append([]byte{}, blockquoteChars...), l.indent...)
			continue
		}

		if listItem, ok := p.(*ast.ListItem); ok {
			// Prepend, as we go from down, but don't count first item.
			l.indent = append(bytes.Repeat(spaceChar, len(listItemMarkerChars(listItem))), l.indent...)
			continue
		}
	}

	// Split whitespace indent from chars.
	cut := bytes.TrimRight(l.indent, noAllocString(spaceChar))
	l.whitespace = l.indent[len(cut):]
	l.indent = cut
}

func (l *lineIndentWriter) AddIndentOnFirstWrite(add []byte) {
	l.firstWriteExtraIndent = append(l.firstWriteExtraIndent, add...)
}

func (l *lineIndentWriter) DelIndentOnFirstWrite(del []byte) {
	l.firstWriteExtraIndent = l.firstWriteExtraIndent[:len(l.firstWriteExtraIndent)-len(del)]
}

func (l *lineIndentWriter) WasIndentOnFirstWriteWritten() bool {
	return len(l.firstWriteExtraIndent) == 0
}

func (l *lineIndentWriter) Write(b []byte) (n int, _ error) {
	if len(b) == 0 {
		return 0, nil
	}

	writtenFromB := 0
	for i, c := range b {
		if l.previousCharWasNewLine {
			ns, err := l.Writer.Write(l.indent)
			n += ns
			if err != nil {
				return n, err
			}
		}

		if c == newLineChar[0] {
			if !l.WasIndentOnFirstWriteWritten() {
				ns, err := l.Writer.Write(l.firstWriteExtraIndent)
				n += ns
				if err != nil {
					return n, err
				}
				l.firstWriteExtraIndent = nil
			}

			ns, err := l.Writer.Write(b[writtenFromB : i+1])
			n += ns
			writtenFromB += ns
			if err != nil {
				return n, err
			}
			l.previousCharWasNewLine = true
			continue
		}

		// Not a newline, make a space if indent was created.
		if l.previousCharWasNewLine && len(l.whitespace) > 0 {
			ns, err := l.Writer.Write(l.whitespace)
			n += ns
			if err != nil {
				return n, err
			}
		}
		l.previousCharWasNewLine = false
	}

	if writtenFromB >= len(b) {
		return n, nil
	}

	if !l.WasIndentOnFirstWriteWritten() {
		ns, err := l.Writer.Write(l.firstWriteExtraIndent)
		n += ns
		if err != nil {
			return n, err
		}
		l.firstWriteExtraIndent = nil
	}

	ns, err := l.Writer.Write(b[writtenFromB:])
	n += ns
	return n, err
}
