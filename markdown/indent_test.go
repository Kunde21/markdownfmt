package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndentationPushes(t *testing.T) {
	tests := []struct {
		desc   string
		pushes []string

		indent string // expected indentation
		ws     string // expected whitespace
	}{
		{
			desc:   "empty",
			indent: "",
			ws:     "",
		},
		{
			desc:   "blockquote",
			pushes: []string{"> "},
			indent: ">",
			ws:     " ",
		},
		{
			desc:   "ws",
			pushes: []string{"    "},
			indent: "",
			ws:     "    ",
		},
		{
			desc:   "ws multiple",
			pushes: []string{"    ", "  ", "    "},
			indent: "",
			ws:     "          ",
		},
		{
			desc:   "ws blockquote",
			pushes: []string{"    ", "> "},
			indent: "    >",
			ws:     " ",
		},
		{
			desc:   "multiple blockquotes",
			pushes: []string{"> ", "> ", "> "},
			indent: "> > >",
			ws:     " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var id indentation
			for _, s := range tt.pushes {
				id.Push([]byte(s))
			}

			assert.Equal(t, tt.indent, string(id.Indent()), "indent")
			assert.Equal(t, tt.ws, string(id.Whitespace()), "whitespace")
		})
	}
}

func TestIndentationPushPop(t *testing.T) {
	t.Run("no-op", func(t *testing.T) {
		var id indentation
		id.Push([]byte("foo"))
		id.Pop()
		assert.Empty(t, id.Indent())
		assert.Empty(t, id.Whitespace())
	})

	t.Run("pop and use", func(t *testing.T) {
		var id indentation
		id.Push([]byte("    "))
		id.Push([]byte("> "))
		id.Pop()

		assert.Equal(t, "", string(id.Indent()))
		assert.Equal(t, "    ", string(id.Whitespace()))
	})

	t.Run("pop and use invert", func(t *testing.T) {
		var id indentation
		id.Push([]byte("> "))
		id.Push([]byte("    "))
		id.Pop()

		assert.Equal(t, ">", string(id.Indent()))
		assert.Equal(t, " ", string(id.Whitespace()))
	})
}

func TestIndentationPopEmpty(t *testing.T) {
	var id indentation
	assert.Panics(t, func() { id.Pop() })
}

func TestTrailingSpaceIdx(t *testing.T) {
	tests := []struct {
		desc string
		give string
		want int
	}{
		{
			desc: "empty",
			give: "",
			want: 0,
		},
		{
			desc: "simple",
			give: "> ",
			want: 1,
		},
		{
			desc: "ws only",
			give: "  ",
			want: 0,
		},
		{
			desc: "non blank only",
			give: ">>>",
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := trailingSpaceIdx([]byte(tt.give))
			assert.Equal(t, tt.want, got)
		})
	}
}
