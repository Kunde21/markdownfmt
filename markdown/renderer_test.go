package markdown

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteClean(t *testing.T) {
	tests := []struct {
		give string
		want string
	}{
		{"foo    bar", "foo bar"},
		{"    ", " "},
		{"foo\n\t\r\nbar", "foo bar"},
		{"foo     ", "foo "},
		{"    foo", " foo"},
	}

	for _, tt := range tests {
		t.Run(tt.give, func(t *testing.T) {
			var buff bytes.Buffer
			require.NoError(t, writeClean(&buff, []byte(tt.give)))
			assert.Equal(t, tt.want, buff.String())
		})
	}
}
