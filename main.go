package main

import (
	"fmt"
	"github.com/r9y9/gossp"
	"github.com/r9y9/gossp/io"
	"github.com/r9y9/gossp/stft"
	"github.com/r9y9/gossp/window"
	"github.com/schollz/progressbar/v3"
	"math"
	"os"
	"strings"
	"sync"
)

// Function to check for errors while reading and writing files
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	// Check arguments
	if len(os.Args) != 2 {
		fmt.Println("Please provide a file: 'go run main.go <filename>.wav'")
		return
	}
	filename := os.Args[1]
	if !strings.HasSuffix(filename, ".wav") {
		// Invalid file format -- Can only process .wav files
		fmt.Println("Please provide a filename in the format 'go run main.go <filename>.wav'")
		return
	}

	fmt.Printf("Attempting to read %s\n", filename)
	r, rerr := io.ReadWav(filename)
	check(rerr)
	data := r.GetMonoData() // Get the mono data from the wav file
	fmt.Println("Successfully mono data from file")

	// Create a Short Time Fourier Transform object to perform the STFT operation
	s := &stft.STFT{
		FrameShift: int(float64(r.SampleRate) / 100.0), // 0.01 sec per frame
		FrameLen:   2048,                               // 2048 samples per frame
		Window:     window.CreateHanning(2048),
	}

	// Compute the STFT of the data and convert it to a gnuplot format
	fmt.Println("Computing STFT...")
	spectrogram, _ := gossp.SplitSpectrogram(s.STFT(data)) // Compute the STFT of the data
	fmt.Println("Successfully computed STFT")
	output := matrixAsGnuplotFormat(&spectrogram) // Convert the STFT to a gnuplot format (x y z)

	// Write the output of the STFT to a file
	writeFileName := fmt.Sprintf("STFT-%s.txt", filename) // <filename>.wav -> STFT-<filename>.wav.txt
	fmt.Printf("Attempting to write STFT values to %s\n", writeFileName)
	w, werr := os.Create(writeFileName)
	check(werr)
	defer w.Close()
	n, werr := w.WriteString(output)
	fmt.Printf("Wrote %d bytes to %s\n", n, writeFileName)
}

// Function to convert the STFT matrix to a gnuplot format (x y z) to be written to a file
// x = time, y = frequency, z = amplitude (logarithmic scale)
// Returns a string in the format "x y z\n" for each individual value in the matrix
// Resulting string can be extremely large for large wav files, sometimes exceeding 1 GB for 3 minute files
func matrixAsGnuplotFormat(matrix *[][]float64) string {
	rows := len(*matrix)
	cols := 2048
	samples := rows * cols
	fmt.Printf("Computing amplitude of %d samples using %d goroutines...\n", samples, rows)

	results := make([]string, rows*cols)           // Preallocate results to ensure correct order
	bar := progressbar.Default(int64(rows * cols)) // Progress bar to show progress of computation

	var wg sync.WaitGroup
	// Mutex for vec is not required -- it isn't being written to
	// Mutex for bar is not required -- Add() is thread-safe

	// Compute the logarithm of each value in the matrix
	for i, vec := range *matrix {
		// Use goroutines to parallelize computation. Decreases computation time *significantly*
		wg.Add(1) // Add this goroutine to the wait group
		go func(i int, vec *[]float64) {
			defer wg.Done() // Mark this goroutine as done when it finishes
			for j := 0; j < cols; j++ {
				index := i*cols + j                                      // Calculate index in results array so the order is preserved
				logVal := math.Log((*vec)[j])                            // Value to be written to file (logarithmic scale)
				results[index] = fmt.Sprintf("%d %d %g\n", i, j, logVal) // Save result in form "<x> <y> <z>\n"
				bar.Add(1)                                               // Increment progress bar
			}
		}(i, &vec)
	}
	wg.Wait() // Wait for all goroutines to finish
	fmt.Printf("All %d goroutines have finished computing amplitude\n", rows)

	// Concatenate results in order
	outputBuilder := strings.Builder{}
	for _, line := range results {
		outputBuilder.WriteString(line)
	}
	return outputBuilder.String()
}
