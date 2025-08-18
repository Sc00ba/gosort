# GoSort

[![Go Test and Coverage](https://github.com/Sc00ba/gosort/actions/workflows/go-test-and-coverage.yml/badge.svg)](https://github.com/Sc00ba/gosort/actions/workflows/go-test-and-coverage.yml)
[![codecov](https://codecov.io/gh/Sc00ba/gosort/branch/main/graph/badge.svg?token=OKOUED3X42)](https://codecov.io/gh/Sc00ba/gosort)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Sc00ba/gosort)](https://golang.org/)
[![License](https://img.shields.io/github/license/Sc00ba/gosort)](LICENSE)

A high-performance, memory-efficient external sorting utility for line-delimited text files in Go. GoSort can handle files larger than available memory by using buffered processing and parallel sorting algorithms.

## Features

- **Memory Efficient**: Processes files using configurable buffer sizes, handling datasets larger than available RAM
- **Parallel Processing**: Utilizes multiple goroutines for concurrent sorting to maximize performance
- **Streaming I/O**: Supports reading from stdin and writing to stdout for pipeline integration
- **Flexible Input/Output**: Works with files or standard streams
- **Token-Based Sorting**: Efficiently sorts line-delimited text using optimized data structures

## Installation

```bash
go install github.com/Sc00ba/gosort/cmd/simple_sort@latest
```

Or clone and build from source:

```bash
git clone https://github.com/Sc00ba/gosort.git
cd gosort
go build -o simple_sort ./cmd/simple_sort
```

## Usage

### Command Line Options

```bash
simple_sort [options]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | stdin | Input file path (use stdin if not specified) |
| `-output` | stdout | Output file path (use stdout if not specified) |
| `-buffer` | 1048576 | Buffer size in bytes (1MB default) |
| `-parallel` | 1 | Number of parallel sorting goroutines |

### Examples

**Sort a file:**
```bash
simple_sort -input data.txt -output sorted.txt
```

**Use in a pipeline:**
```bash
cat unsorted.txt | simple_sort > sorted.txt
```

**Sort with custom buffer size and parallel processing:**
```bash
simple_sort -input large_file.txt -output sorted.txt -buffer 4194304 -parallel 4
```

**Sort from stdin to stdout:**
```bash
echo -e "zebra\napple\nbanana" | simple_sort
```

## How It Works

GoSort implements an external sorting algorithm optimized for line-delimited text:

1. **Buffering**: Reads data into a configurable buffer size
2. **Token Extraction**: Identifies line boundaries and creates lightweight token references
3. **Parallel Sorting**: Splits tokens across multiple goroutines for concurrent sorting
4. **Heap Merging**: Uses a min-heap to efficiently merge sorted segments
5. **Streaming Output**: Writes sorted results without storing the entire dataset in memory

The implementation uses compact data structures to minimize memory overhead while maintaining high performance.

## Limitations

- Input must be line-delimited text (using `\n` as delimiter)
- Maximum buffer size is capped at 4GB
- Partial tokens at buffer boundaries are trimmed (not processed)
- Files without line delimiters cannot be processed

## Development

### Running Tests

```bash
go test ./...
```

### Running Tests with Coverage

```bash
go test -v -coverprofile=coverage.txt -covermode=atomic ./...
```

### Project Structure

```
gosort/
├── cmd/simple_sort/     # Command-line application
├── internal/sort/       # Core sorting algorithms and data structures
├── .github/workflows/   # CI/CD pipeline
└── go.mod              # Go module definition
```

## Requirements

- Go 1.24.0 or later
- Unix-like system (tested on Linux/macOS)

## License

See the LICENSE file

## Acknowledgments

- Uses Go's standard library `container/heap` for efficient merging
- Inspired by classic external sorting algorithms
