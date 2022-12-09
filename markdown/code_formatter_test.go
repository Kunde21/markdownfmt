package markdown

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFormatGo(t *testing.T) {
	tests := []struct {
		desc string
		give string
		want string
	}{
		{
			desc: "empty",
			give: "",
			want: "",
		},
		{
			desc: "valid code",
			give: "func main(){fmt.Println(msg)\n}",
			want: "func main() {\n\tfmt.Println(msg)\n}",
		},
		{
			desc: "invalid code",
			give: "func main(){",
			want: "func main(){",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := formatGo([]byte(tt.give))
			if diff := cmp.Diff(tt.want, string(got)); len(diff) > 0 {
				t.Errorf("formatGo(%q) = %q, want %q\ndiff %v", tt.give, string(got), tt.want, diff)
			}
		})
	}
}
