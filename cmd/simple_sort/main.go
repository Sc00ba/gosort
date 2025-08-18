package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"gosort/internal/sort"
)

func main() {
	inputFile := flag.String("input", "", "Input file path")
	outputFile := flag.String("output", "", "Output file path")
	bufferSize := flag.Int("buffer", 1024*1024, "Buffer size in bytes")
	parallel := flag.Int("parallel", 1, "Number of parallel sorters")
	flag.Parse()

	var reader io.Reader
	if *inputFile == "" {
		reader = os.Stdin
	} else {
		file, err := os.Open(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			return
		}
		defer file.Close()
		reader = file
	}

	var writer io.Writer
	if *outputFile == "" {
		writer = os.Stdout
	} else {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return
		}
		defer file.Close()
		writer = file
	}

	buffer, trimmed, err := sort.NewBuffer(reader, *bufferSize)
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error creating buffer: %v\n", err)
		return
	}

	if len(trimmed) > 0 {
		fmt.Fprintf(os.Stderr, "input is %d bytes larger than the buffer\n", len(trimmed))
		return
	}

	it := buffer.Sort(*parallel)

	br := bufio.NewWriterSize(writer, 4*1024*1024)
	defer br.Flush()
	for {
		token, ok := it.NextToken()
		if !ok {
			break
		}
		_, err := br.Write(token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to output: %v\n", err)
			return
		}
	}
}
