package chunks

import (
	"bytes"
	"context"
	"sort"
	"sync"
)

type sorterOptions struct {
	descending bool
}

type sorterOption func(options *sorterOptions)

// NewSorter reads chunks from the given channel and sorts chunks by lexicographical order and writes them to the output channel.
// The order is ascending by default.
// Any errors that occur during the sorting process will be sent on the returned errs channel.
func NewSorter(ctx context.Context, numSorters int, chunks <-chan [][]byte, options ...sorterOption) (<-chan [][]byte, <-chan error, error) {
	wg := &sync.WaitGroup{}
	out := make(chan [][]byte)
	errs := make(chan error, 1)
	for range numSorters {
		wg.Add(1)
		go func(chunks <-chan [][]byte, out chan<- [][]byte, errs chan<- error) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				case chunk, ok := <-chunks:
					if !ok {
						return
					}

					sort.Slice(chunk, func(i, j int) bool {
						return bytes.Compare(chunk[i], chunk[j]) < 0
					})

					if err := send(ctx, out, chunk); err != nil {
						errs <- err
						return
					}
				}
			}
		}(chunks, out, errs)
	}

	go func() {
		wg.Wait()
		close(out)
		close(errs)
	}()

	return out, errs, nil
}
