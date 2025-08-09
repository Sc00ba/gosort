package chunks

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChunker(t *testing.T) {
	makeChunks := func(chunks [][]string) [][][]byte {
		var result [][][]byte
		for _, chunk := range chunks {
			var byteChunk [][]byte
			for _, line := range chunk {
				byteChunk = append(byteChunk, []byte(line))
			}
			result = append(result, byteChunk)
		}
		return result
	}

	tests := []struct {
		name           string
		inputs         []string
		chunkSize      uint
		bufferSize     uint
		expectedChunks [][][]byte
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Basic chunking with single input",
			inputs:         []string{"line1\nline2\nline3\nline4"},
			chunkSize:      10,
			bufferSize:     2,
			expectedChunks: makeChunks([][]string{{"line1", "line2"}, {"line3", "line4"}}),
			expectError:    false,
		},
		{
			name:           "Multiple inputs basic chunking",
			inputs:         []string{"line1\nline2", "line3\nline4"},
			chunkSize:      10,
			bufferSize:     2,
			expectedChunks: makeChunks([][]string{{"line1", "line2"}, {"line3", "line4"}}),
			expectError:    false,
		},
		{
			name:           "Input fits in a single chunk",
			inputs:         []string{"hello\nworld"},
			chunkSize:      20,
			bufferSize:     1,
			expectedChunks: makeChunks([][]string{{"hello", "world"}}),
			expectError:    false,
		},
		{
			name:           "Empty input",
			inputs:         []string{""},
			chunkSize:      10,
			bufferSize:     1,
			expectedChunks: makeChunks([][]string{}),
			expectError:    false,
		},
		{
			name:           "Token larger than chunk size",
			inputs:         []string{"this line is just too long for the chunk size"},
			chunkSize:      10,
			bufferSize:     1,
			expectedChunks: nil,
			expectError:    true,
			errorContains:  "error encountered while scanning",
		},
		{
			name:           "Token size equals chunk size",
			inputs:         []string{"0123456789"},
			chunkSize:      10,
			bufferSize:     1,
			expectedChunks: makeChunks([][]string{{"0123456789"}}),
			expectError:    false,
		},
		{
			name:          "Token size one more than chunk size",
			inputs:        []string{"0123456789A"},
			chunkSize:     10,
			bufferSize:    1,
			expectError:   true,
			errorContains: "error encountered while scanning",
		},
		{
			name:           "Input is an exact fit for a chunk",
			inputs:         []string{"lineA\nlineB"},
			chunkSize:      10,
			bufferSize:     1,
			expectedChunks: makeChunks([][]string{{"lineA", "lineB"}}),
			expectError:    false,
		},
		{
			name:           "Input with no trailing newline",
			inputs:         []string{"line1\nline2"},
			chunkSize:      10,
			bufferSize:     1,
			expectedChunks: makeChunks([][]string{{"line1", "line2"}}),
			expectError:    false,
		},
		{
			name:           "Input with empty lines",
			inputs:         []string{"line1\n\nline3"},
			chunkSize:      10,
			bufferSize:     1,
			expectedChunks: makeChunks([][]string{{"line1", "", "line3"}}),
			expectError:    false,
		},
		{
			name:           "Chunking with a single large token that fits",
			inputs:         []string{"onelongline"},
			chunkSize:      12,
			bufferSize:     1,
			expectedChunks: makeChunks([][]string{{"onelongline"}}),
			expectError:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var readers []io.Reader
			for _, input := range tc.inputs {
				readers = append(readers, strings.NewReader(input))
			}

			chunksChan, errs, err := NewChunker(ctx, tc.chunkSize, tc.bufferSize, readers...)
			assert.NoError(t, err, "NewChunker should not return an immediate error")

			var wg sync.WaitGroup
			wg.Add(2)

			var actualChunks [][][]byte
			go func() {
				defer wg.Done()
				for chunk := range chunksChan {
					actualChunks = append(actualChunks, chunk)
				}
			}()

			var receivedErr error
			go func() {
				defer wg.Done()
				for err := range errs {
					if err != nil {
						receivedErr = err
					}
				}
			}()

			wg.Wait()

			if tc.expectError {
				assert.Error(t, receivedErr, "Expected an error but got none")
				if tc.errorContains != "" {
					assert.Contains(t, receivedErr.Error(), tc.errorContains, "Error message does not contain expected text")
				}
			} else {
				assert.NoError(t, receivedErr, "Received an unexpected error")
				assert.Equal(t, tc.expectedChunks, actualChunks, "The received chunks do not match the expected chunks")
			}
		})
	}
}
