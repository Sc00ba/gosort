package main

import (
	"context"
	"flag"
	"gosort/internal/chunks"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"golang.org/x/sys/unix"
)

const (
	defaultMemoryPercent = 0.8
)

var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	outFileArg := flag.String("o", "", "Write result to a file instead of standard output.")
	bufferSizeArg := flag.Uint("S", 0, "use SIZE for main memory buffer.")
	parallel := flag.Uint("parallel", 0, "change the number of sorts to N")

	flag.Parse()

	runSort(outFileArg, bufferSizeArg, parallel)

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		log.Printf("✅ Memory profile written to %s\n", *memprofile)
	}
}

func runSort(outFileArg *string, bufferSizeArg *uint, parallel *uint) {
	var openedFiles []*os.File
	defer func() {
		for _, file := range openedFiles {
			_ = file.Close()
		}
	}()

	filePaths := flag.Args()
	readFromStdIn := len(filePaths) == 0 || filePaths[0] == "-"

	outFile := os.Stdout
	if *outFileArg != "" {
		file, err := os.OpenFile(*outFileArg, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("failed to open out file %s (%s)\n", *outFileArg, err)
		}

		outFile = file
		openedFiles = append(openedFiles, outFile)
	}

	var info unix.Sysinfo_t
	err := unix.Sysinfo(&info)
	if err != nil {
		log.Fatalf("Error calling sysinfo: %s", err)
	}

	totalMemory := uint(float64(info.Totalram * uint64(info.Unit)))
	if *bufferSizeArg > totalMemory {
		log.Fatalf("-S argument %d is too large\n", *bufferSizeArg)
	}

	bufferSize := uint(float64(info.Totalram*uint64(info.Unit)) * defaultMemoryPercent)
	if *bufferSizeArg > 0 {
		bufferSize = *bufferSizeArg
	}

	numSorters := uint(runtime.NumCPU())
	if *parallel > 0 {
		numSorters = *parallel
	}

	chunkSize := bufferSize / numSorters / 2 // Split the bufferSize between the chunks channel and the sorter routines.
	totalChunks := bufferSize / chunkSize
	channelBufferSize := totalChunks - numSorters

	var readers []io.Reader
	if readFromStdIn {
		readers = append(readers, os.Stdin)
	} else {
		log.Fatal("stdin is the only input file currently supported")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chunksChan, chunkerErrs, err := chunks.NewChunker(ctx, chunkSize, channelBufferSize, readers...)
	if err != nil {
		log.Fatalf("error while creating chunker (%s)", err)
	}

	sortedChunksChan, sortErrs, err := chunks.NewSorter(ctx, int(numSorters), chunksChan)
	if err != nil {
		log.Fatalf("error while creating sorter (%s)", err)
	}

	mergeErrs, err := chunks.NewMerger(ctx, sortedChunksChan, outFile, 100)
	if err != nil {
		log.Fatalf("error while creating merger (%s)", err)
	}

	run := true
	for run {
		select {
		case err, ok := <-chunkerErrs:
			if ok {
				log.Fatal(err)
			}
		case err, ok := <-sortErrs:
			if ok {
				log.Fatal(err)
			}
		case err, ok := <-mergeErrs:
			if ok {
				log.Fatal(err)
			} else {
				run = false
			}
		}
	}
}
