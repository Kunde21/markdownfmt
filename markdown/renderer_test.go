package markdown

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
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
			if err := writeClean(&buff, []byte(tt.give)); err != nil {
				t.Fatal(err)
			}

			got := buff.String()
			if diff := cmp.Diff(tt.want, got); len(diff) > 0 {
				t.Errorf("clean(%q) = %q, want %q", tt.give, got, tt.want)
			}
		})
	}
}

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
		if err := writeClean(&buff, []byte(s)); err != nil {
			t.Fatal(err)
		}
		got := buff.String()

		if diff := cmp.Diff(string(want), got); len(diff) > 0 {
			t.Errorf("clean(%q) = %q, want %q", s, got, want)
		}
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
