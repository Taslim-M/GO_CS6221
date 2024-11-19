package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/r9y9/gossp"
	"github.com/r9y9/gossp/io"
	"github.com/r9y9/gossp/stft"
	"github.com/r9y9/gossp/window"
	"github.com/schollz/progressbar/v3"

	"bufio"
	"image"
	"image/color"
	"image/png"
	"strconv"
)

func main() {
	router := gin.Default()

	// Serve index.html at the default route
	router.GET("/", func(c *gin.Context) {
		c.File("./index.html")
	})

	// Handle POST request to process file and return base64 image
	router.POST("/", func(c *gin.Context) {
		// Get the filename from the form data
		filename := c.PostForm("filename")
		if filename == "" {
			c.String(http.StatusBadRequest, "Filename is required")
			return
		}
		fmt.Printf("Attempting to read %s\n", filename)
		// Simulate processing and return a sample base64-encoded image for demonstration
		// Replace this logic with actual MFCC processing and image generation
		perform_mfcc(filename)
		dummyImage, _ := ioutil.ReadFile("image.png")
		base64Image := base64.StdEncoding.EncodeToString(dummyImage)

		c.String(http.StatusOK, base64Image)
	})

	router.Run(":8080") // Start server on port 8080
}

// Function to check for errors while reading and writing files
func check(e error) {
	if e != nil {
		panic(e)
	}
}

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
		//FrameShift: int(float64(r.SampleRate) / 50.0), // 20 ms per frame)
		//FrameLen:   1024,                              // 1024 samples per frame (adjustable)
		//Window:     window.CreateHanning(1024),
	}

	// Compute the STFT of the data and convert it to a gnuplot format
	fmt.Println("Computing STFT...")
	fmt.Println(float64(r.SampleRate))
	spectrogram, _ := gossp.SplitSpectrogram(s.STFT(data)) // Compute the STFT of the data
	fmt.Println("Successfully computed STFT")
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

	// Parse the STFT data from the file
	stft, err := parseSTFT("image.txt") //STFT-Short.wav.txt
	if err != nil {
		fmt.Println("Error reading STFT file:", err)
		return
	}

	//checking size of the STFT data
	fmt.Printf("STFT dimensions: %d rows, %d columns\n", len(stft), len(stft[0]))

	//Set Parameters
	numMelFilters := 40 // Mel filter banks
	numMFCC := 13       // # of MFCCs to return

	// Step 1: Apply Mel filterbank
	melSpectrogram := applyMelFilterbank(stft, numMelFilters)

	// Step 2: Log transform
	logMelSpectrogram := logMelSpectrogram(melSpectrogram)

	// Step 3: Compute MFCCs
	mfcc := computeMFCC(logMelSpectrogram, numMFCC)

	fmt.Println("Generated MFCC dimensions:", len(mfcc), "rows,", len(mfcc[0]), "columns")

	//Generate spectrogram using gnuplot
	//generateSpectrogram(mfccFile, "MFCC_Spectrogram.png")

	// Step 4: Downsample MFCC so generated image is visible
	downsampleFactor := 100 // Adjust as needed
	mfccDownsampled := downsampleMatrix(mfcc, downsampleFactor)
	fmt.Println("Downsampled MFCC dimensions:", len(mfccDownsampled), "rows,", len(mfccDownsampled[0]), "columns")

	// Saving MFCC data to a text file
	mfccFile := "MFCC_data.txt"
	if err := saveMFCCToFile(mfccDownsampled, mfccFile); err != nil {
		fmt.Println("Error saving MFCC data:", err)
		return
	}
	fmt.Printf("MFCC data saved to %s\n", mfccFile)

	//was getting exit error 1 when I try to integrate gnuplot into this file. I ran plot.gp separately in terminal using the command go run main.go
	//In the spectogram, y axis = instances of mfcc coefficients, x axis = time frames, color scale = magnitute of mfcc coefficients
}

// Save the MFCC matrix to a file for Gnuplot
func saveMFCCToFile(matrix [][]float64, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, row := range matrix {
		for j, val := range row {
			if j > 0 {
				file.WriteString(" ")
			}
			file.WriteString(fmt.Sprintf("%f", val))
		}
		file.WriteString("\n")
	}
	return nil
}

// Generate a spectrogram using Gnuplot
func generateSpectrogram(dataFile, outputFile string) {
	gnuplotScript := `
        set terminal pngcairo size 1024,768
        set output "` + outputFile + `"
        set xlabel "Time (frames)"
        set ylabel "MFCC Coefficients"
        set title "MFCC Spectrogram"
        set palette defined (0 "blue", 1 "cyan", 2 "green", 3 "yellow", 4 "red")
        unset key
        plot "` + dataFile + `" matrix with image
    `

	cmd := exec.Command("gnuplot", "-e", gnuplotScript)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error generating spectrogram with gnuplot:", err)
		return
	}
	fmt.Println("MFCC Spectrogram saved to", outputFile)
}

// Downsamples a 2D matrix by taking every nth row
func downsampleMatrix(matrix [][]float64, factor int) [][]float64 {
	if factor <= 1 {
		return matrix // No downsampling needed
	}

	var downsampled [][]float64
	for i := 0; i < len(matrix); i += factor {
		downsampled = append(downsampled, matrix[i])
	}
	return downsampled
}

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

// Function to parse the STFT.txt file
func parseSTFT(filename string) ([][]float64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var stft [][]float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		values := strings.Fields(line)

		// Skip empty rows
		if len(values) == 0 {
			continue
		}

		row := make([]float64, len(values))
		for i, v := range values {
			val, _ := strconv.ParseFloat(v, 64)
			row[i] = val
		}
		stft = append(stft, row)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return stft, nil
}

// Function to apply Mel filterbank
func applyMelFilterbank(stft [][]float64, numMelFilters int) [][]float64 {
	if len(stft) == 0 || len(stft[0]) == 0 {
		panic("STFT data is empty or improperly formatted")
	}

	numColumns := len(stft[0]) // Number of columns in STFT
	melSpectrogram := make([][]float64, len(stft))

	for i := range stft {
		// Ensure the row is not empty
		if len(stft[i]) == 0 {
			continue
		}

		melSpectrogram[i] = make([]float64, numMelFilters)
		for j := 0; j < numMelFilters; j++ {
			melSpectrogram[i][j] = stft[i][j%numColumns] // Loop around if `numMelFilters` > `numColumns`
		}
	}
	return melSpectrogram
}

// Function to compute logarithm of Mel spectrogram
func logMelSpectrogram(melSpectrogram [][]float64) [][]float64 {
	logMel := make([][]float64, len(melSpectrogram))
	for i := range melSpectrogram {
		logMel[i] = make([]float64, len(melSpectrogram[i]))
		for j := range melSpectrogram[i] {
			logMel[i][j] = math.Log(melSpectrogram[i][j] + 1e-10)
		}
	}
	return logMel
}

// Custom Discrete Cosine Transform (DCT) for MFCC calculation
func dctTransform(input []float64, numCoefficients int) []float64 {
	N := len(input)
	output := make([]float64, numCoefficients)
	for k := 0; k < numCoefficients; k++ {
		sum := 0.0
		for n := 0; n < N; n++ {
			sum += input[n] * math.Cos(math.Pi/float64(N)*(float64(n)+0.5)*float64(k))
		}
		output[k] = sum
	}
	return output
}

// Function to compute MFCCs using custom DCT
func computeMFCC(logMelSpectrogram [][]float64, numCoefficients int) [][]float64 {
	mfcc := make([][]float64, len(logMelSpectrogram))
	for i := range logMelSpectrogram {
		mfcc[i] = dctTransform(logMelSpectrogram[i], numCoefficients)
	}
	return mfcc
}

// Function to save MFCC as an image with proper scaling
// Function to save MFCC as a heatmap with proper scaling
func saveMFCCImage(mfcc [][]float64, filename string) error {
	height := len(mfcc)   // Number of frames (time axis)
	width := len(mfcc[0]) // Number of MFCC coefficients (coefficient axis)

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Find min and max values of MFCC to normalize the range
	var minVal, maxVal float64
	minVal = math.Inf(1)
	maxVal = math.Inf(-1)

	// Determine the min and max values of the MFCC
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			val := mfcc[y][x]
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	// Scale and map MFCC values to a color gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			normalizedValue := (mfcc[y][x] - minVal) / (maxVal - minVal)
			// Map to RGB gradient: Blue (low) to Red (high)
			colorValue := color.RGBA{
				R: uint8(normalizedValue * 255),       // High values = red
				G: uint8((1 - normalizedValue) * 255), // Mid-range values= green
				B: uint8((1 - normalizedValue) * 255), // Low values = blue
				A: 255,
			}
			img.Set(x, y, colorValue)
		}
	}

	// Save the image to a PNG file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}
