package chunks

import (
	"context"
)

func send[T any](ctx context.Context, ch chan<- T, value T) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- value:
		return nil
	}
}
