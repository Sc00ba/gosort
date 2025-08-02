package chunks

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSorter(t *testing.T) {
	makeByteChunks := func(stringChunks [][]string) [][][]byte {
		var result [][][]byte
		for _, chunk := range stringChunks {
			var byteChunk [][]byte
			for _, line := range chunk {
				byteChunk = append(byteChunk, []byte(line))
			}
			result = append(result, byteChunk)
		}
		return result
	}

	tests := []struct {
		name        string
		numSorters  int
		inputChunks [][][]byte
		expectError bool
	}{
		{
			name:       "Single chunk, single sorter",
			numSorters: 1,
			inputChunks: makeByteChunks([][]string{
				{"zebra", "apple", "monkey", "banana"},
			}),
			expectError: false,
		},
		{
			name:       "Multiple chunks, single sorter",
			numSorters: 1,
			inputChunks: makeByteChunks([][]string{
				{"zebra", "apple"},
				{"monkey", "banana"},
				{"cat", "dog"},
			}),
			expectError: false,
		},
		{
			name:       "Multiple chunks, multiple sorters",
			numSorters: 4,
			inputChunks: makeByteChunks([][]string{
				{"zebra", "apple"},
				{"monkey", "banana"},
				{"cat", "dog"},
				{"yak", "fish", "gorilla"},
			}),
			expectError: false,
		},
		{
			name:        "Empty input channel",
			numSorters:  2,
			inputChunks: makeByteChunks([][]string{}),
			expectError: false,
		},
		{
			name:       "Chunks with empty strings",
			numSorters: 1,
			inputChunks: makeByteChunks([][]string{
				{"line1", "", "line3"},
			}),
			expectError: false,
		},
		{
			name:       "Already sorted chunk",
			numSorters: 1,
			inputChunks: makeByteChunks([][]string{
				{"apple", "banana", "cherry"},
			}),
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			chunksIn := make(chan [][]byte, len(tc.inputChunks))

			sortedChunks, errs, err := NewSorter(ctx, tc.numSorters, chunksIn)
			assert.NoError(t, err, "NewSorter should not return an immediate error")

			go func() {
				for _, chunk := range tc.inputChunks {
					chunksIn <- chunk
				}
				close(chunksIn)
			}()

			var wg sync.WaitGroup
			wg.Add(2)

			var receivedErr error
			go func() {
				defer wg.Done()
				for err := range errs {
					if err != nil {
						receivedErr = err
					}
				}
			}()

			var actualSortedChunks [][]string
			go func() {
				defer wg.Done()
				for chunk := range sortedChunks {
					var strChunk []string
					for _, tokenBytes := range chunk {
						strChunk = append(strChunk, string(tokenBytes))
					}
					actualSortedChunks = append(actualSortedChunks, strChunk)
				}
			}()

			wg.Wait()

			if tc.expectError {
				assert.Error(t, receivedErr)
				return
			}

			assert.NoError(t, receivedErr, "Received an unexpected error")

			var expectedSortedChunks [][]string
			for _, chunk := range tc.inputChunks {
				var stringChunk []string
				for _, line := range chunk {
					stringChunk = append(stringChunk, string(line))
				}
				sort.Strings(stringChunk)
				expectedSortedChunks = append(expectedSortedChunks, stringChunk)
			}

			sort.Slice(expectedSortedChunks, func(i, j int) bool {
				return strings.Join(expectedSortedChunks[i], "") < strings.Join(expectedSortedChunks[j], "")
			})
			sort.Slice(actualSortedChunks, func(i, j int) bool {
				return strings.Join(actualSortedChunks[i], "") < strings.Join(actualSortedChunks[j], "")
			})

			assert.Equal(t, expectedSortedChunks, actualSortedChunks, "The set of sorted files does not match the set of sorted input chunks")
		})
	}
}
