package main

import (
	"context"
	"flag"
	"gosort/internal/chunks"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"

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
			log.Printf("could not create memory profile (%s)", err)
			return
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Printf("could not write memory profile (%s)", err)
			return
		}
		log.Printf("✅ Memory profile written to %s\n", *memprofile)
	}
}

func runSort(outFileArg *string, bufferSizeArg *uint, parallel *uint) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

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
			log.Printf("Failed to open out file %s (%s)\n", *outFileArg, err)
			return
		}

		outFile = file
		openedFiles = append(openedFiles, outFile)
	}

	var info unix.Sysinfo_t
	err := unix.Sysinfo(&info)
	if err != nil {
		log.Printf("Error calling sysinfo: %s", err)
		return
	}

	totalMemory := uint(float64(info.Totalram * uint64(info.Unit)))
	if *bufferSizeArg > totalMemory {
		log.Printf("-S argument %d is too large\n", *bufferSizeArg)
		return
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
		log.Printf("Stdin is the only input file currently supported")
		return
	}

	var wg sync.WaitGroup

	chunksChan, chunkerErrs, err := chunks.NewChunker(ctx, chunkSize, channelBufferSize, readers...)
	if err != nil {
		cancel()
		log.Printf("Error while creating chunker (%s)", err)
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range chunkerErrs {
			log.Printf("Error from chunker (%s)", err)
			cancel()
		}
	}()

	sortedChunksChan, sortErrs, err := chunks.NewSorter(ctx, int(numSorters), chunksChan)
	if err != nil {
		cancel()
		log.Printf("Error while creating sorter (%s)", err)
		wg.Wait()
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range sortErrs {
			log.Printf("Error from sorter (%s)", err)
			cancel()
		}
	}()

	mergeFactor := 100
	mergeErrs, err := chunks.NewMerger(ctx, sortedChunksChan, outFile, mergeFactor)
	if err != nil {
		cancel()
		log.Printf("Error while creating merger (%s)", err)
		wg.Wait()
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range mergeErrs {
			log.Printf("Error from merger (%s)", err)
			cancel()
		}
	}()

	wg.Wait()
}
