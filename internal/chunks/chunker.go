package chunks

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
)

const (
	chunkAllocation = 1024
)

// NewChunker creates a chunker that reads from a set of readers and chunks tokens. Tokens are separated by newlines.
// chunkSize is in bytes. If the size of a token exceeds the chunkSize processing will stop and an error will be sent on the errs channel.
// A chunks channel is returned with the buffer set to the given bufferSize.
func NewChunker(ctx context.Context, chunkSize, bufferSize uint, readers ...io.Reader) (<-chan [][]byte, <-chan error, error) {
	out := make(chan [][]byte, bufferSize)
	errs := make(chan error, 1)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(readers []io.Reader, out chan<- [][]byte) {
		defer wg.Done()
		for _, reader := range readers {
			scanner := bufio.NewScanner(reader)
			chunk := make([][]byte, 0, chunkAllocation)
			var currentSize uint
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				default:
					token := make([]byte, len(scanner.Bytes()))
					copy(token, scanner.Bytes())
					tokenSize := uint(len(token))
					if tokenSize > chunkSize {
						errs <- fmt.Errorf("token size greater than chunk size")
						return
					}

					if currentSize+tokenSize <= chunkSize {
						chunk = append(chunk, token)
						currentSize += tokenSize
					} else {
						if err := send(ctx, out, chunk); err != nil {
							errs <- err
							return
						}
						chunk = make([][]byte, 0, chunkAllocation)
						chunk = append(chunk, token)
						currentSize = tokenSize
					}
				}
			}
			if err := scanner.Err(); err != nil {
				errs <- err
				return
			}

			if len(chunk) > 0 {
				if err := send(ctx, out, chunk); err != nil {
					errs <- err
					return
				}
			}
		}
	}(readers, out)

	go func() {
		wg.Wait()
		close(out)
		close(errs)
	}()

	return out, errs, nil
}
