package tcellterm

// import (
// 	"strings"
// 	"testing"
//
// 	"github.com/stretchr/testify/assert"
// )
//
// func TestParseOSC8(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		input    string
// 		expected string
// 	}{
// 		{
// 			name:     "no semicolon in URI",
// 			input:    "8;;https://example.com",
// 			expected: "https://example.com",
// 		},
// 		{
// 			name:     "no semicolon in URI, with id",
// 			input:    "8;id=hello;https://example.com",
// 			expected: "https://example.com",
// 		},
// 		{
// 			name:     "semicolon in URI",
// 			input:    "8;;https://example.com/semi;colon",
// 			expected: "https://example.com/semi;colon",
// 		},
// 		{
// 			name:     "multiple semicolons in URI",
// 			input:    "8;;https://example.com/s;e;m;i;colon",
// 			expected: "https://example.com/s;e;m;i;colon",
// 		},
// 		{
// 			name:     "semicolon in URI, with id",
// 			input:    "8;id=hello;https://example.com/semi;colon",
// 			expected: "https://example.com/semi;colon",
// 		},
// 		{
// 			name:     "terminating sequence",
// 			input:    "8;;",
// 			expected: "",
// 		},
// 		{
// 			name:     "terminating sequence with id",
// 			input:    "8;id=hello;",
// 			expected: "",
// 		},
// 	}
//
// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			uri := parseOSC8(strings.Split(test.input, ";"))
// 			assert.Equal(t, test.expected, uri)
// 		})
// 	}
// }
