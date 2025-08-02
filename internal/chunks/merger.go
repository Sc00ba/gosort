package chunks

import (
	"bufio"
	"bytes"
	"container/heap"
	"context"
	"io"
	"os"
)

type mergeOptions struct {
	descending bool
}

type mergeOption func(opts *mergeOptions)

func NewMerger(ctx context.Context, chunks <-chan [][]byte, out io.Writer, threshold int, options ...mergeOption) (<-chan error, error) {
	errs := make(chan error, 1)
	go func() {
		defer close(errs)
		err := Merge(ctx, chunks, out, threshold, options...)
		if err != nil {
			errs <- err
		}
	}()

	return errs, nil
}

// Merge performs a K-way merge on the sorted chunks and will use temporary files if the given
// threshold is exceeded.
func Merge(ctx context.Context, chunks <-chan [][]byte, out io.Writer, threshold int, options ...mergeOption) error {
	var tmpFiles []*os.File
	defer func() {
		for _, file := range tmpFiles {
			file.Close()
			os.Remove(file.Name())
		}
	}()

	dir := "/tmp"
	batch := make([][][]byte, 0, threshold)
	run := true
	for run {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-chunks:
			if !ok {
				run = false
				break
			}
			if len(batch) < threshold {
				batch = append(batch, chunk)
			} else {
				tmpFile, err := os.CreateTemp(dir, "gosort")
				if err != nil {
					return err
				}

				writer := bufio.NewWriter(tmpFile)
				err = mergeBatchToWriter(ctx, batch, writer)
				if err != nil {
					return err
				}

				err = writer.Flush()
				if err != nil {
					return err
				}

				tmpFiles = append(tmpFiles, tmpFile)
				batch = make([][][]byte, 0, threshold)
				batch = append(batch, chunk)
			}
		}
	}

	if len(tmpFiles) > 0 {
		tmpFile, err := os.CreateTemp(dir, "gosort")
		if err != nil {
			return err
		}

		err = mergeBatchToWriter(ctx, batch, tmpFile)
		if err != nil {
			return err
		}

		tmpFiles = append(tmpFiles, tmpFile)

		err = mergeFilesToWriter(ctx, tmpFiles, out)
		if err != nil {
			return err
		}
	} else {
		err := mergeBatchToWriter(ctx, batch, out)
		if err != nil {
			return err
		}
	}

	return nil
}

func mergeBatchToWriter(ctx context.Context, batch [][][]byte, writer io.Writer) error {
	mergeHeap := make(mergeHeap, 0, len(batch))
	for i := range batch {
		heap.Push(
			&mergeHeap,
			mergeStep{
				srcIdx:   i,
				tokenIdx: 0,
				token:    batch[i][0],
			})
	}

	for mergeHeap.Len() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			mergeStep := heap.Pop(&mergeHeap).(mergeStep)
			_, err := writer.Write(mergeStep.token)
			if err != nil {
				return err
			}

			_, err = writer.Write([]byte{'\n'})
			if err != nil {
				return err
			}

			mergeStep.tokenIdx++
			if mergeStep.tokenIdx < len(batch[mergeStep.srcIdx]) {
				mergeStep.token = batch[mergeStep.srcIdx][mergeStep.tokenIdx]
				heap.Push(&mergeHeap, mergeStep)
			}
		}
	}

	return nil
}

func mergeFilesToWriter(ctx context.Context, files []*os.File, writer io.Writer) error {
	scanners := make([]*bufio.Scanner, len(files))
	mergeHeap := make(mergeHeap, 0, len(files))
	for i := range files {
		scanner := bufio.NewScanner(files[i])
		scanners[i] = scanner

		if scanners[i].Scan() {
			token := make([]byte, len(scanner.Bytes()))
			copy(token, scanner.Bytes())
			heap.Push(
				&mergeHeap,
				mergeStep{
					srcIdx: i,
					token:  token,
				})
		}
	}

	for mergeHeap.Len() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			mergeStep := heap.Pop(&mergeHeap).(mergeStep)
			_, err := writer.Write(mergeStep.token)
			if err != nil {
				return err
			}

			_, err = writer.Write([]byte{'\n'})
			if err != nil {
				return err
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

	return nil
}

type mergeStep struct {
	srcIdx   int
	tokenIdx int
	token    []byte
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
