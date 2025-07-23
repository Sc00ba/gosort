package chunks

import (
	"bufio"
	"bytes"
	"container/heap"
	"context"
	"fmt"
	"io"
)

type mergeOptions struct {
	descending bool
}

type mergeOption func(opts *mergeOptions)

// Merge performs a K-way merge on the given token readers where tokens are separated by newlines.
// Readers are expected to output tokens in a lexicographic order. The default order is ascending.
func Merge(ctx context.Context, readers []io.Reader, out io.Writer, options ...mergeOption) error {
	writer := bufio.NewWriter(out)

	var scanners []*bufio.Scanner
	for _, reader := range readers {
		scanners = append(scanners, bufio.NewScanner(reader))
	}

	mergeHeap := make(mergeHeap, 0, len(scanners))
	for idx, scanner := range scanners {
		select {
		case <-ctx.Done():
			return nil
		default:
			if scanner.Scan() {
				token := make([]byte, len(scanner.Bytes()))
				copy(token, scanner.Bytes())
				heap.Push(&mergeHeap, mergeStep{srcIdx: idx, token: token})
			}
		}
	}

	for mergeHeap.Len() > 0 {
		select {
		case <-ctx.Done():
			return nil
		default:
			mergeStep := heap.Pop(&mergeHeap).(mergeStep)
			_, err := writer.Write(mergeStep.token)
			if err != nil {
				return fmt.Errorf("failed to write token (%w)", err)
			}

			err = writer.WriteByte('\n')
			if err != nil {
				return fmt.Errorf("failed to write newline (%w)", err)
			}

			scanner := scanners[mergeStep.srcIdx]
			if scanner.Scan() {
				token := make([]byte, len(scanner.Bytes()))
				copy(token, scanner.Bytes())
				mergeStep.token = token
				heap.Push(&mergeHeap, mergeStep)
			}
		}
	}

	return writer.Flush()
}

type mergeStep struct {
	srcIdx int
	token  []byte
}

type mergeHeap []mergeStep

func (h *mergeHeap) Len() int {
	return len(*h)
}

func (h *mergeHeap) Less(i, j int) bool {
	return bytes.Compare((*h)[i].token, (*h)[j].token) < 0
}

func (h *mergeHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *mergeHeap) Push(x any) {
	*h = append(*h, x.(mergeStep))
}

func (h *mergeHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
