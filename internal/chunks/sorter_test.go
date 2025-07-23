package chunks

import (
	"bufio"
	"context"
	"os"
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

			tmpFilesChan, errs, err := NewSorter(ctx, tc.numSorters, chunksIn)
			assert.NoError(t, err, "NewSorter should not return an immediate error")

			go func() {
				for _, chunk := range tc.inputChunks {
					chunksIn <- chunk
				}
				close(chunksIn)
			}()

			var wg sync.WaitGroup
			wg.Add(2)

			var tmpFiles []string
			go func() {
				defer wg.Done()
				for f := range tmpFilesChan {
					tmpFiles = append(tmpFiles, f)
					t.Cleanup(func() {
						os.Remove(f)
					})
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
				assert.Error(t, receivedErr)
				return
			}

			assert.NoError(t, receivedErr, "Received an unexpected error")
			assert.Equal(t, len(tc.inputChunks), len(tmpFiles), "Should have one output file per input chunk")

			var expectedSortedChunks [][]string
			for _, chunk := range tc.inputChunks {
				var stringChunk []string
				for _, line := range chunk {
					stringChunk = append(stringChunk, string(line))
				}
				sort.Strings(stringChunk)
				expectedSortedChunks = append(expectedSortedChunks, stringChunk)
			}

			var actualSortedChunks [][]string
			for _, fileName := range tmpFiles {
				file, err := os.Open(fileName)
				assert.NoError(t, err, "Failed to open temp file for verification")
				var lines []string
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				file.Close()
				actualSortedChunks = append(actualSortedChunks, lines)
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
