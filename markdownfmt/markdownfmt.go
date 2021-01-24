package markdownfmt

import (
	"bytes"
	"io/ioutil"

	"github.com/Kunde21/markdownfmt/v2/markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
)

func NewGoldmark(opts ...renderer.Option) goldmark.Markdown {
	mr := markdown.NewRenderer()
	extensions := []goldmark.Extender{
		extension.GFM,
	}
	parserOptions := []parser.Option{
		parser.WithAttribute(), // We need this to enable # headers {#custom-ids}.
	}

	gm := goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(parserOptions...),
	)
	// Set renderer outside constructor to reset the
	// html render functions registered by GFM.
	gm.SetRenderer(mr)
	gm.Renderer().AddOptions(opts...)

	return gm
}

// Process formats given Markdown.
func Process(filename string, src []byte, opts ...renderer.Option) ([]byte, error) {
	text, err := readSource(filename, src)
	if err != nil {
		return nil, err
	}

	output := bytes.NewBuffer(nil)
	if err := NewGoldmark(opts...).Convert(text, output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// If src != nil, readSource returns src.
// If src == nil, readSource returns the result of reading the file specified by filename.
func readSource(filename string, src []byte) ([]byte, error) {
	if src != nil {
		return src, nil
	}
	return ioutil.ReadFile(filename)
}
