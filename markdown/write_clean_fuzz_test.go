//go:build go1.18
// +build go1.18

package markdown

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzWriteClean verifies parity between
// cleanWithoutTrim and the new writeClean function.
func FuzzWriteClean(f *testing.F) {
	f.Add("foo    bar")
	f.Add("    ")
	f.Add("foo\n\t\r\nbar")
	f.Add("foo     ")
	f.Add("    foo")

	f.Fuzz(func(t *testing.T, s string) {
		want := string(cleanWithoutTrim([]byte(s)))

		var buff bytes.Buffer
		require.NoError(t, writeClean(&buff, []byte(s)))
		assert.Equal(t, want, buff.String())
	})
}

// cleanWithoutTrim is an oler version of writeClean
// retained in tests to verify parity of the new implementation.
func cleanWithoutTrim(b []byte) []byte {
	var ret []byte
	var p byte
	for i := 0; i < len(b); i++ {
		q := b[i]
		if q == '\n' || q == '\r' || q == '\t' {
			q = ' '
		}
		if q != ' ' || p != ' ' {
			ret = append(ret, q)
			p = q
		}
	}
	return ret
}
