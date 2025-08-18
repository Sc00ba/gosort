package sort

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuffer(t *testing.T) {
	type inputs struct {
		reader io.Reader
		size   int
	}
	type expectations struct {
		trimmed []byte
		err     error
	}
	type testCase struct {
		name  string
		build func() (inputs, expectations)
	}

	testCases := []testCase{
		{
			name: "partial_token_read",
			build: func() (inputs, expectations) {
				s := "a\nb\nc\n123..."
				r := strings.NewReader(s)
				trimmed := []byte("123...")
				return inputs{
						reader: r,
						size:   r.Len(),
					}, expectations{
						trimmed: trimmed,
						err:     nil,
					}
			},
		},
		{
			name: "no_token_delimiter",
			build: func() (inputs, expectations) {
				s := "0123456789..."
				r := strings.NewReader(s)
				return inputs{
						reader: r,
						size:   r.Len() / 2,
					}, expectations{
						trimmed: nil,
						err:     ErrorTokenDelimiterNotFound,
					}
			},
		},
		{
			name: "size_too_big",
			build: func() (inputs, expectations) {
				s := "a\nb\nc\n"
				r := strings.NewReader(s)
				return inputs{
						reader: r,
						size:   maxBufferSize + 1,
					}, expectations{
						trimmed: nil,
						err:     ErrorBufferSizeTooBig,
					}
			},
		},
		{
			name: "empty_line",
			build: func() (inputs, expectations) {
				s := "\n"
				r := strings.NewReader(s)
				return inputs{
						reader: r,
						size:   1024,
					}, expectations{
						trimmed: []byte{},
						err:     nil,
					}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, expects := tc.build()
			_, gotTrimmed, gotErr := NewBuffer(in.reader, in.size)
			assert.ErrorIs(t, gotErr, expects.err)
			assert.Equal(t, expects.trimmed, gotTrimmed)
		})
	}
}

func TestBuffer_Sort(t *testing.T) {
	type inputs struct {
		reader   io.Reader
		size     int
		parallel int
	}
	type expectations struct {
		tokens [][]byte
	}
	type testCase struct {
		name  string
		build func() (inputs, expectations)
	}

	testCases := []testCase{
		{
			name: "one_sorters",
			build: func() (inputs, expectations) {
				s := "i\nh\ng\nf\ne\nd\nc\nb\na\n"
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'a', '\n'},
					{'b', '\n'},
					{'c', '\n'},
					{'d', '\n'},
					{'e', '\n'},
					{'f', '\n'},
					{'g', '\n'},
					{'h', '\n'},
					{'i', '\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len(),
						parallel: 1,
					}, expectations{
						tokens: tokens,
					}
			},
		},
		{
			name: "buffer_smaller_than_input",
			build: func() (inputs, expectations) {
				s := "a\nb\nc\na\nb\nc\na\nb\nc\n"
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'a', '\n'},
					{'b', '\n'},
					{'c', '\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len() / 3,
						parallel: 1,
					}, expectations{
						tokens: tokens,
					}
			},
		},
		{
			name: "buffer_bigger_than_input",
			build: func() (inputs, expectations) {
				s := "i\nh\ng\nf\ne\nd\nc\nb\na\n"
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'a', '\n'},
					{'b', '\n'},
					{'c', '\n'},
					{'d', '\n'},
					{'e', '\n'},
					{'f', '\n'},
					{'g', '\n'},
					{'h', '\n'},
					{'i', '\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len() * 3,
						parallel: 1,
					}, expectations{
						tokens: tokens,
					}
			},
		},
		{
			name: "two_sorters",
			build: func() (inputs, expectations) {
				s := "i\nh\ng\nf\ne\nd\nc\nb\na\n"
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'a', '\n'},
					{'b', '\n'},
					{'c', '\n'},
					{'d', '\n'},
					{'e', '\n'},
					{'f', '\n'},
					{'g', '\n'},
					{'h', '\n'},
					{'i', '\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len(),
						parallel: 2,
					}, expectations{
						tokens: tokens,
					}
			},
		},
		{
			name: "more_sorters_than_tokens",
			build: func() (inputs, expectations) {
				s := "i\nh\ng\nf\ne\nd\nc\nb\na\n"
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'a', '\n'},
					{'b', '\n'},
					{'c', '\n'},
					{'d', '\n'},
					{'e', '\n'},
					{'f', '\n'},
					{'g', '\n'},
					{'h', '\n'},
					{'i', '\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len(),
						parallel: len(tokens) * 2,
					}, expectations{
						tokens: tokens,
					}
			},
		},
		{
			name: "parallel_is_zero",
			build: func() (inputs, expectations) {
				s := "i\nh\ng\nf\ne\nd\nc\nb\na\n"
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'a', '\n'},
					{'b', '\n'},
					{'c', '\n'},
					{'d', '\n'},
					{'e', '\n'},
					{'f', '\n'},
					{'g', '\n'},
					{'h', '\n'},
					{'i', '\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len(),
						parallel: 0,
					}, expectations{
						tokens: tokens,
					}
			},
		},
		{
			name: "partial_token_trimmed",
			build: func() (inputs, expectations) {
				s := "i\nh\ng\nf\ne\nd\nc\nb\na\n123..."
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'a', '\n'},
					{'b', '\n'},
					{'c', '\n'},
					{'d', '\n'},
					{'e', '\n'},
					{'f', '\n'},
					{'g', '\n'},
					{'h', '\n'},
					{'i', '\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len(),
						parallel: 2,
					}, expectations{
						tokens: tokens,
					}
			},
		},
		{
			name: "empty_lines",
			build: func() (inputs, expectations) {
				s := "\n\n\n"
				r := strings.NewReader(s)
				tokens := [][]byte{
					{'\n'},
					{'\n'},
					{'\n'},
				}
				return inputs{
						reader:   r,
						size:     r.Len(),
						parallel: 2,
					}, expectations{
						tokens: tokens,
					}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, expects := tc.build()
			b, _, err := NewBuffer(in.reader, in.size)
			require.NoError(t, err)

			var gotTokens [][]byte
			it := b.Sort(in.parallel)
			for {
				token, ok := it.NextToken()
				if !ok {
					break
				}

				gotTokens = append(gotTokens, token)
			}

			assert.Equal(t, expects.tokens, gotTokens)
		})
	}
}
