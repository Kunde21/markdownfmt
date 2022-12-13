package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.want, string(got))
		})
	}
}
