package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/r9y9/gossp"
	"github.com/r9y9/gossp/io"
	"github.com/r9y9/gossp/stft"
	"github.com/r9y9/gossp/window"
	"github.com/schollz/progressbar/v3"
)

func perform_mfcc(filename string) {

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
	spec_init := s.STFT(data)
	spectrogram, _ := gossp.SplitSpectrogram(spec_init) // Compute the STFT of the data
	fmt.Println("Successfully computed STFT")
	// Reconstruct
	reconstructed := s.ISTFT(spec_init)
	//Save the real and inverse audio data
	WriteMono("curr.wav", data, r.SampleRate)
	WriteMono("recreate.wav", reconstructed, r.SampleRate)
	// Generate the plot

	if err := createLineChart(data, reconstructed, "chart.png"); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Line chart saved as chart.png")
	}
	output := matrixAsGnuplotFormat(&spectrogram) // Convert the STFT to a gnuplot format (x y z)

	// Write the output of the STFT to a file
	// writeFileName := fmt.Sprintf("STFT-%s.txt", filename) // <filename>.wav -> STFT-<filename>.wav.txt
	writeFileName := fmt.Sprintf("image.txt") // <filename>.wav -> STFT-<filename>.wav.txt
	fmt.Printf("Attempting to write STFT values to %s\n", writeFileName)
	w, werr := os.Create(writeFileName)
	check(werr)
	defer w.Close()
	n, werr := w.WriteString(output)
	fmt.Printf("Wrote %d bytes to %s\n", n, writeFileName)

	// Create gnuplot of STFT values
	createGnuplot(writeFileName)
}

// Function to create a heatmap using Gnuplot
// Requires the filename of the file containing the STFT values created by matrixAsGnuplotFormat
// Creates a heatmap of the STFT values and saves it as a .png file
func createGnuplot(filename string) {
	// Replace .txt with .png for output file name
	plotFile := strings.Replace(filename, ".txt", ".png", 1)

	// Create a temporary file for the Gnuplot script
	scriptFile, err := os.CreateTemp("", "gnuplot-script-*.gp")
	if err != nil {
		log.Fatalf("Failed to create temporary script file: %v", err)
	}
	defer os.Remove(scriptFile.Name()) // Clean up the script file after running Gnuplot

	// Write the Gnuplot script to the temporary file
	// For some reason gnuplot prefers using a temp file instead of the already written file
	scriptContent := fmt.Sprintf(`
set terminal png size 800,600
set output "%s"
set view map
set xlabel "X"
set ylabel "Y"
set cblabel "Z"
set palette rgbformulae 33,13,10
set pm3d interpolate 0,0
splot "%s" using 1:2:3 with pm3d notitle
`, plotFile, filename)
	if _, err := scriptFile.WriteString(scriptContent); err != nil {
		log.Fatalf("Failed to write to temporary script file: %v", err)
	}
	scriptFile.Close() // Ensure content is flushed to disk

	// Execute Gnuplot with the script file
	fmt.Println("Creating heatmap using Gnuplot...")
	bar := progressbar.Default(-1, "Creating Gnuplot heatmap")
	cmd := exec.Command("gnuplot", scriptFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		bar.Finish()
		log.Fatalf("Failed to run Gnuplot: %v", err)
	}
	bar.Finish()
	fmt.Printf("Heatmap generated and saved as %s\n", plotFile)
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

	results := make([]string, rows*cols)                            // Preallocate results to ensure correct order
	bar := progressbar.Default(int64(rows*cols), "Performing STFT") // Progress bar to show progress of computation

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

	// Concatenate results with blank lines between rows
	outputBuilder := strings.Builder{}
	for i, line := range results {
		outputBuilder.WriteString(line)
		if (i+1)%cols == 0 {
			outputBuilder.WriteString("\n") // Add blank line after each row
		}
	}
	return outputBuilder.String()
}
