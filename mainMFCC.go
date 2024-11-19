package main

// import (
// 	"encoding/base64"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"math"
// 	"net/http"
// 	"os"
// 	"os/exec"
// 	"strings"
// 	"sync"

// 	"github.com/gin-gonic/gin"
// 	"github.com/r9y9/gossp"
// 	"github.com/r9y9/gossp/io"
// 	"github.com/r9y9/gossp/stft"
// 	"github.com/r9y9/gossp/window"
// 	"github.com/schollz/progressbar/v3"
// )

// func main() {
// 	router := gin.Default()

// 	// Serve index.html at the default route
// 	router.GET("/", func(c *gin.Context) {
// 		c.File("./index.html")
// 	})

// 	// Handle POST request to process file and return base64 image
// 	router.POST("/", func(c *gin.Context) {
// 		// Get the filename from the form data
// 		filename := c.PostForm("filename")
// 		if filename == "" {
// 			c.String(http.StatusBadRequest, "Filename is required")
// 			return
// 		}
// 		fmt.Printf("Attempting to read %s\n", filename)
// 		// Simulate processing and return a sample base64-encoded image for demonstration
// 		// Replace this logic with actual MFCC processing and image generation
// 		perform_mfcc(filename)
// 		dummyImage, _ := ioutil.ReadFile("image.png")
// 		base64Image := base64.StdEncoding.EncodeToString(dummyImage)

// 		c.String(http.StatusOK, base64Image)
// 	})

// 	router.Run(":8080") // Start server on port 8080
// }

// // Function to check for errors while reading and writing files
// func check(e error) {
// 	if e != nil {
// 		panic(e)
// 	}
// }

// func perform_mfcc(filename string) {

// 	if !strings.HasSuffix(filename, ".wav") {
// 		// Invalid file format -- Can only process .wav files
// 		fmt.Println("Please provide a filename in the format 'go run main.go <filename>.wav'")
// 		return
// 	}

// 	fmt.Printf("Attempting to read %s\n", filename)
// 	r, rerr := io.ReadWav(filename)
// 	check(rerr)
// 	data := r.GetMonoData() // Get the mono data from the wav file
// 	fmt.Println("Successfully mono data from file")

// 	// Create a Short Time Fourier Transform object to perform the STFT operation
// 	s := &stft.STFT{
// 		FrameShift: int(float64(r.SampleRate) / 100.0), // 0.01 sec per frame
// 		FrameLen:   2048,                               // 2048 samples per frame
// 		Window:     window.CreateHanning(2048),
// 	}

// 	// Compute the STFT of the data and convert it to a gnuplot format
// 	fmt.Println("Computing STFT...")
// 	spectrogram, _ := gossp.SplitSpectrogram(s.STFT(data)) // Compute the STFT of the data
// 	powerSpectrogram := powerSpectrum(spectrogram)
// 	fmt.Println("Successfully computed STFT")
// 	output := matrixAsGnuplotFormat(&spectrogram) // Convert the STFT to a gnuplot format (x y z)

// 	// Write the output of the STFT to a file
// 	// writeFileName := fmt.Sprintf("STFT-%s.txt", filename) // <filename>.wav -> STFT-<filename>.wav.txt
// 	writeFileName := fmt.Sprintf("image.txt") // <filename>.wav -> STFT-<filename>.wav.txt
// 	fmt.Printf("Attempting to write STFT values to %s\n", writeFileName)
// 	w, werr := os.Create(writeFileName)
// 	check(werr)
// 	defer w.Close()
// 	n, werr := w.WriteString(output)
// 	fmt.Printf("Wrote %d bytes to %s\n", n, writeFileName)

// 	// Create gnuplot of STFT values
// 	createGnuplot(writeFileName)
// }

// // Function to create a heatmap using Gnuplot
// // Requires the filename of the file containing the STFT values created by matrixAsGnuplotFormat
// // Creates a heatmap of the STFT values and saves it as a .png file
// func createGnuplot(filename string) {
// 	// Replace .txt with .png for output file name
// 	plotFile := strings.Replace(filename, ".txt", ".png", 1)

// 	// Create a temporary file for the Gnuplot script
// 	scriptFile, err := os.CreateTemp("", "gnuplot-script-*.gp")
// 	if err != nil {
// 		log.Fatalf("Failed to create temporary script file: %v", err)
// 	}
// 	defer os.Remove(scriptFile.Name()) // Clean up the script file after running Gnuplot

// 	// Write the Gnuplot script to the temporary file
// 	// For some reason gnuplot prefers using a temp file instead of the already written file
// 	scriptContent := fmt.Sprintf(`
// set terminal png size 800,600
// set output "%s"
// set view map
// set xlabel "X"
// set ylabel "Y"
// set cblabel "Z"
// set palette rgbformulae 33,13,10
// set pm3d interpolate 0,0
// splot "%s" using 1:2:3 with pm3d notitle
// `, plotFile, filename)
// 	if _, err := scriptFile.WriteString(scriptContent); err != nil {
// 		log.Fatalf("Failed to write to temporary script file: %v", err)
// 	}
// 	scriptFile.Close() // Ensure content is flushed to disk

// 	// Execute Gnuplot with the script file
// 	fmt.Println("Creating heatmap using Gnuplot...")
// 	bar := progressbar.Default(-1, "Creating Gnuplot heatmap")
// 	cmd := exec.Command("gnuplot", scriptFile.Name())
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	if err := cmd.Run(); err != nil {
// 		bar.Finish()
// 		log.Fatalf("Failed to run Gnuplot: %v", err)
// 	}
// 	bar.Finish()
// 	fmt.Printf("Heatmap generated and saved as %s\n", plotFile)
// }

// // Function to convert the STFT matrix to a gnuplot format (x y z) to be written to a file
// // x = time, y = frequency, z = amplitude (logarithmic scale)
// // Returns a string in the format "x y z\n" for each individual value in the matrix
// // Resulting string can be extremely large for large wav files, sometimes exceeding 1 GB for 3 minute files
// func matrixAsGnuplotFormat(matrix *[][]float64) string {
// 	rows := len(*matrix)
// 	cols := 2048
// 	samples := rows * cols
// 	fmt.Printf("Computing amplitude of %d samples using %d goroutines...\n", samples, rows)

// 	results := make([]string, rows*cols)                            // Preallocate results to ensure correct order
// 	bar := progressbar.Default(int64(rows*cols), "Performing STFT") // Progress bar to show progress of computation

// 	var wg sync.WaitGroup
// 	// Mutex for vec is not required -- it isn't being written to
// 	// Mutex for bar is not required -- Add() is thread-safe

// 	// Compute the logarithm of each value in the matrix
// 	for i, vec := range *matrix {
// 		// Use goroutines to parallelize computation. Decreases computation time *significantly*
// 		wg.Add(1) // Add this goroutine to the wait group
// 		go func(i int, vec *[]float64) {
// 			defer wg.Done() // Mark this goroutine as done when it finishes
// 			for j := 0; j < cols; j++ {
// 				index := i*cols + j                                      // Calculate index in results array so the order is preserved
// 				logVal := math.Log((*vec)[j])                            // Value to be written to file (logarithmic scale)
// 				results[index] = fmt.Sprintf("%d %d %g\n", i, j, logVal) // Save result in form "<x> <y> <z>\n"
// 				bar.Add(1)                                               // Increment progress bar
// 			}
// 		}(i, &vec)
// 	}
// 	wg.Wait() // Wait for all goroutines to finish
// 	fmt.Printf("All %d goroutines have finished computing amplitude\n", rows)

// 	// Concatenate results with blank lines between rows
// 	outputBuilder := strings.Builder{}
// 	for i, line := range results {
// 		outputBuilder.WriteString(line)
// 		if (i+1)%cols == 0 {
// 			outputBuilder.WriteString("\n") // Add blank line after each row
// 		}
// 	}
// 	return outputBuilder.String()
// }

// // ---------------------------------------------------------------------------------------------------------------------------

// // Helper function to convert frequency to Mel scale
// func hzToMel(hz float64) float64 {
// 	return 2595 * math.Log10(1+hz/700)
// }

// // Helper function to convert Mel scale to frequency
// func melToHz(mel float64) float64 {
// 	return 700 * (math.Pow(10, mel/2595) - 1)
// }

// // Helper function to linearly interpolate between two values
// func interpolate(x, x1, x2, y1, y2 float64) float64 {
// 	return y1 + (y2-y1)*(x-x1)/(x2-x1)
// }

// // Mel Filter Bank Creation
// func createMelFilterBank(numFilters, fftSize, sampleRate int) [][]float64 {
// 	// Initialize the filter bank
// 	melFilters := make([][]float64, numFilters)

// 	// Compute the Mel frequency range
// 	minHz := 0.0
// 	maxHz := float64(sampleRate) / 2.0 // Nyquist frequency
// 	minMel := hzToMel(minHz)
// 	maxMel := hzToMel(maxHz)

// 	// Compute the Mel points
// 	melPoints := make([]float64, numFilters+2) // Include start and end
// 	for i := 0; i < len(melPoints); i++ {
// 		melPoints[i] = minMel + (maxMel-minMel)*float64(i)/float64(len(melPoints)-1)
// 	}

// 	// Convert Mel points to Hertz
// 	hzPoints := make([]float64, len(melPoints))
// 	for i, mel := range melPoints {
// 		hzPoints[i] = melToHz(mel)
// 	}

// 	// Map Hertz to FFT bin indices
// 	binPoints := make([]int, len(hzPoints))
// 	for i, hz := range hzPoints {
// 		binPoints[i] = int(math.Floor((fftSize + 1) * hz / float64(sampleRate)))
// 	}

// 	// Create triangular filters
// 	for i := 1; i <= numFilters; i++ {
// 		melFilters[i-1] = make([]float64, fftSize/2+1)
// 		for j := binPoints[i-1]; j < binPoints[i]; j++ {
// 			melFilters[i-1][j] = interpolate(float64(j), float64(binPoints[i-1]), float64(binPoints[i]), 0, 1)
// 		}
// 		for j := binPoints[i]; j < binPoints[i+1]; j++ {
// 			melFilters[i-1][j] = interpolate(float64(j), float64(binPoints[i]), float64(binPoints[i+1]), 1, 0)
// 		}
// 	}

// 	return melFilters
// }

// // Power Spectrum Computation
// func powerSpectrum(spectrogram [][]complex128) [][]float64 {
// 	powerSpectrogram := make([][]float64, len(spectrogram))
// 	for i, frame := range spectrogram {
// 		powerSpectrogram[i] = make([]float64, len(frame))
// 		for j, value := range frame {
// 			powerSpectrogram[i][j] = real(value)*real(value) + imag(value)*imag(value) // |X|^2
// 		}
// 	}
// 	return powerSpectrogram
// }

// // Log Mel Spectrum Computation
// func logMelSpectrum(powerSpectrogram [][]float64, melFilterBank [][]float64) [][]float64 {
// 	numFrames := len(powerSpectrogram)
// 	numFilters := len(melFilterBank)
// 	melSpectrogram := make([][]float64, numFrames)

// 	for i := 0; i < numFrames; i++ {
// 		melSpectrogram[i] = make([]float64, numFilters)
// 		for j, filter := range melFilterBank {
// 			sum := 0.0
// 			for k, value := range powerSpectrogram[i] {
// 				sum += value * filter[k]
// 			}
// 			melSpectrogram[i][j] = math.Log(sum + 1e-10) // Logarithmic compression
// 		}
// 	}
// 	return melSpectrogram
// }

// // Discrete Cosine Transform (DCT)
// func dct(logMelSpectrogram [][]float64, numCoefficients int) [][]float64 {
// 	numFrames := len(logMelSpectrogram)
// 	mfcc := make([][]float64, numFrames)

// 	for i, logMelFrame := range logMelSpectrogram {
// 		mfcc[i] = make([]float64, numCoefficients)
// 		for k := 0; k < numCoefficients; k++ {
// 			sum := 0.0
// 			for n, value := range logMelFrame {
// 				sum += value * math.Cos(float64(k)*(float64(n)+0.5)*math.Pi/float64(len(logMelFrame)))
// 			}
// 			mfcc[i][k] = sum
// 		}
// 	}
// 	return mfcc
// }
