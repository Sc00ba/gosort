package chunks

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
)

type sorterOptions struct {
	descending bool
}

type sorterOption func(options *sorterOptions)

// NewSorter reads chunks from the given channel and sorts chunks by lexicographical order and writes them to temporary files.
// The order is ascending by default.
// The temporary file names are sent on the returned channel.
// Any errors that occur during the sorting process will be sent on the returned errs channel.
func NewSorter(ctx context.Context, numSorters int, chunks <-chan [][]byte, options ...sorterOption) (<-chan string, <-chan error, error) {
	wg := &sync.WaitGroup{}

	tmpFiles := make(chan string)
	errs := make(chan error, 1)
	for range numSorters {
		wg.Add(1)
		go func(chunks <-chan [][]byte, tmpFiles chan<- string, errs chan<- error) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-chunks:
					if !ok {
						return
					}

					sort.Slice(chunk, func(i, j int) bool {
						return bytes.Compare(chunk[i], chunk[j]) < 0
					})

					tmpFile, err := os.CreateTemp("/tmp", "gosort")
					if err != nil {
						errs <- fmt.Errorf("failed to create temp file (%w)", err)
					}

					writer := bufio.NewWriter(tmpFile)

					for i := range chunk {
						_, err := writer.Write(chunk[i])
						if err != nil {
							errs <- fmt.Errorf("failed to write to tmp file (%w)", err)
							return
						}

						err = writer.WriteByte('\n')
						if err != nil {
							errs <- fmt.Errorf("failed to write newline to tmp file (%w)", err)
							return
						}
					}

					err = writer.Flush()
					if err != nil {
						errs <- fmt.Errorf("failed to flush writer (%w)", err)
						_ = tmpFile.Close()
						return
					}

					err = tmpFile.Close()
					if err != nil {
						errs <- fmt.Errorf("failed to close tmp file (%w)", err)
						return
					}

					tmpFiles <- tmpFile.Name()
				}
			}
		}(chunks, tmpFiles, errs)
	}

	go func(tmpFiles chan string, errs chan error) {
		wg.Wait()
		close(tmpFiles)
		close(errs)
	}(tmpFiles, errs)

	return tmpFiles, errs, nil
}
