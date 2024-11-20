package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/r9y9/gossp"
	"github.com/r9y9/gossp/io"
	"github.com/r9y9/gossp/stft"
	"github.com/r9y9/gossp/window"
)

func perform_stft_pipeline(filename string) {

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
	createLineChart(data, reconstructed, "chart.png")

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

// Function to check for errors while reading and writing files
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func perform_stft_standalone(filename string) {

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
	start := time.Now()
	spectrogram, _ := gossp.SplitSpectrogram(s.STFT(data)) // Compute the STFT of the data
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
	fmt.Println("Successfully computed STFT")
	fmt.Print(len(spectrogram))
}
