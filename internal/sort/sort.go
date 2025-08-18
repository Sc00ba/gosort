package sort

import (
	"bytes"
	"container/heap"
	"errors"
	"io"
	"sort"
	"sync"
)

var (
	ErrorTokenDelimiterNotFound      = errors.New("token delimiter not found")
	ErrorBufferSizeTooBig            = errors.New("buffer size too big")
	tokenDelim                  byte = '\n'
	tokenSizeGuess                   = 8
	maxBufferSize                    = 4 * 1024 * 1024 * 1024
	minSplitSize                     = 1024
	minParallel                      = 1
	maxParallel                      = 8
)

type TokenIterator interface {
	NextToken() ([]byte, bool)
}

type info uint64

const infoMask = uint64(0x0FFFFFFFF)

func newInfo(offset, length uint32) info {
	return info(uint64(offset)<<32 | uint64(length))
}

func (i info) offset() uint32 {
	return uint32((uint64(i) &^ infoMask) >> 32)
}

func (i info) len() uint32 {
	return uint32(uint64(i) & infoMask)
}

type buffer struct {
	data  []byte
	infos []info
}

func NewBuffer(r io.Reader, size int) (*buffer, []byte, error) {
	if size > maxBufferSize {
		return nil, nil, ErrorBufferSizeTooBig
	}

	data := make([]byte, size)
	n, err := io.ReadFull(r, data)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, nil, err
	}

	b := &buffer{
		data: data[:n],
	}

	b.computeOffsets()

	trimmed, err := b.trim()
	if err != nil {
		return nil, nil, err
	}

	return b, trimmed, nil
}

func (b *buffer) trim() ([]byte, error) {
	var delimIdx int
	var found bool
	for i := len(b.data) - 1; i >= 0; i-- {
		if b.data[i] == tokenDelim {
			found = true
			delimIdx = i
			break
		}
	}

	if !found {
		return nil, ErrorTokenDelimiterNotFound
	}

	lastOffset := delimIdx + 1

	trimmed := make([]byte, len(b.data)-lastOffset)
	copy(trimmed, b.data[lastOffset:])
	b.data = b.data[:lastOffset]
	return trimmed, nil
}

func (b *buffer) computeOffsets() {
	b.infos = make([]info, 0, len(b.data)/tokenSizeGuess)
	offset := uint32(0)
	for i := range b.data {
		if b.data[i] == tokenDelim {
			b.infos = append(b.infos, newInfo(offset, uint32(i)+1-offset))
			offset = uint32(i + 1)
		}
	}
}

func (b *buffer) Sort(parallel int) TokenIterator {
	parallel = max(minParallel, min(maxParallel, parallel))
	for parallel > 1 && len(b.infos)/parallel < minSplitSize {
		parallel--
	}
	n := len(b.infos) / parallel

	var splits [][]info
	for i := range parallel {
		j := i * n
		var split []info
		if i < parallel-1 {
			split = b.infos[j : j+n]
		} else {
			split = b.infos[j:]
		}

		if len(split) > 0 {
			splits = append(splits, split)
		}
	}

	wg := &sync.WaitGroup{}
	for _, s := range splits {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sort.Sort(&sortSplit{data: b.data, infos: s})
		}()
	}

	wg.Wait()

	h := &splitHeap{data: b.data, heap: splits}
	heap.Init(h)

	return h
}

type sortSplit struct {
	data  []byte
	infos []info
}

func (s *sortSplit) Len() int {
	return len(s.infos)
}

func (s *sortSplit) Less(i, j int) bool {
	iInfo := s.infos[i]
	iO := iInfo.offset()
	iN := iInfo.len()
	jInfo := s.infos[j]
	jO := jInfo.offset()
	jN := jInfo.len()
	return bytes.Compare(s.data[iO:iO+iN], s.data[jO:jO+jN]) < 0
}

func (s *sortSplit) Swap(i, j int) {
	s.infos[i], s.infos[j] = s.infos[j], s.infos[i]
}

type splitHeap struct {
	data []byte
	heap [][]info
}

func (h *splitHeap) Len() int {
	return len(h.heap)
}

func (h *splitHeap) Less(i, j int) bool {
	iInfo := h.heap[i][0]
	iO := iInfo.offset()
	iN := iInfo.len()
	jInfo := h.heap[j][0]
	jO := jInfo.offset()
	jN := jInfo.len()
	return bytes.Compare(h.data[iO:iO+iN], h.data[jO:jO+jN]) < 0
}

func (h *splitHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
}

func (h *splitHeap) Push(split any) {
	h.heap = append(h.heap, split.([]info))
}

func (h *splitHeap) Pop() any {
	n := len(h.heap)
	split := h.heap[n-1]
	h.heap = h.heap[0 : n-1]
	return split
}

func (h *splitHeap) NextToken() ([]byte, bool) {
	if h.Len() == 0 {
		return nil, false
	}

	infos := heap.Pop(h).([]info)
	info := infos[0]
	offset := info.offset()
	n := info.len()
	token := h.data[offset : offset+n]

	if len(infos) > 1 {
		heap.Push(h, infos[1:])
	}

	return token, true
}
