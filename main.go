package main

import (
	"fmt"
	"github.com/go-audio/wav"
	"log"
	"math/rand"
	"os"
)

func main() {
	// Open the WAV file
	fmt.Println("Reading WAV file...")
	file, err := os.Open("Song.wav")
	if err != nil {
		log.Fatal("Failed to open WAV file!", err)
	}
	defer file.Close()

	// Create the decoder with the opened file
	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		log.Fatal("Invalid WAV file!")
	}

	// Decode samples into a buffer
	fmt.Println("Decoding samples...")
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatal("Failed to decode samples!", err)
	}

	fmt.Println("Start processing...")
	samples := make([]float64, len(buffer.Data))
	for i, sample := range buffer.Data {
		// I tried using goroutines here, but the main bottleneck is the decoding process
		// It doesn't make any meaningful difference to parallelize this part.
		// TBH the compiler probably optimizes this already
		samples[i] = float64(sample) / float64(1<<15) // Normalize to [-1, 1]
	}

	// Print sample data for verification
	fmt.Println("Number of samples:", len(samples))

	var randomSamples []float64
	for i := 0; i < 10; i++ {
		// Collect a random collection of samples to display
		randInt := rand.Int() % len(samples)
		randomSamples = append(randomSamples, samples[randInt])
	}

	fmt.Println("Random selection of samples:", randomSamples)
}
