package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Kunde21/markdownfmt/v2/markdownfmt"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type metaRender struct{}

// RegisterFuncs ...
func (m metaRender) RegisterFuncs(r renderer.NodeRendererFuncRegisterer) {
	r.Register(meta.KindMetadata, renderMeta)
}

func renderMeta(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	m, ok := n.(*meta.Metadata)
	if !ok {
		fmt.Fprintf(w, "%v", n)
	}
	fmt.Fprintln(w, "---------------------")
	for _, v := range m.Items {
		fmt.Fprintf(w, "%s: %s\n", v.Key, v.Value)
	}
	fmt.Fprintln(w, "---------------------")
	return ast.WalkContinue, nil
}

func TestMeta(t *testing.T) {
	mdfmt := markdownfmt.NewGoldmark()
	meta.New().Extend(mdfmt)
	mdfmt.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(metaRender{}, 500),
		),
	)
	source := `---
Title: goldmark-meta
Summary: Add YAML metadata to the document
Tags:
    - markdown
    - goldmark
---

# Hello goldmark-meta
`

	var buf bytes.Buffer
	if err := mdfmt.Convert([]byte(source), &buf); err != nil {
		panic(err)
	}
	fmt.Print(buf.String())
}
