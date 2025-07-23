package chunks

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string // Each string MUST represent a pre-sorted "file"
	}{
		{
			name: "Two simple inputs",
			inputs: []string{
				"apple\nbanana\nzebra",
				"cat\ndog\nmonkey",
			},
		},
		{
			name: "Three inputs",
			inputs: []string{
				"a\nc\ne",
				"b\nd",
				"f\ng\nh",
			},
		},
		{
			name: "Inputs with overlapping values",
			inputs: []string{
				"apple\ncherry\ndate",
				"banana\ncherry\nfig",
			},
		},
		{
			name: "One input is empty",
			inputs: []string{
				"a\nb\nc",
				"",
				"d\ne",
			},
		},
		{
			name:   "Single input source",
			inputs: []string{"alpha\nbeta\ngamma"},
		},
		{
			name:   "All inputs are empty",
			inputs: []string{"", "", ""},
		},
		{
			name: "Inputs with varying lengths",
			inputs: []string{
				"1\n10\n100",
				"2\n3\n4\n5\n6\n7",
				"8",
			},
		},
		{
			name: "Inputs with empty lines",
			inputs: []string{
				"\n\n\na\nc",
				"b",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var readers []io.Reader
			var allInputLines []string

			for _, inputStr := range tc.inputs {
				readers = append(readers, strings.NewReader(inputStr))
				if inputStr != "" {
					lines := strings.Split(inputStr, "\n")
					allInputLines = append(allInputLines, lines...)
				}
			}

			sort.Strings(allInputLines)
			expectedOutput := strings.Join(allInputLines, "\n")
			if expectedOutput != "" {
				expectedOutput += "\n"
			}

			var outputBuffer bytes.Buffer
			err := Merge(ctx, readers, &outputBuffer)

			assert.NoError(t, err, "Merge function returned an unexpected error")
			assert.Equal(t, expectedOutput, outputBuffer.String(), "The merged output does not match the expected sorted output")
		})
	}
}
