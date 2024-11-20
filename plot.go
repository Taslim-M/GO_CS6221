package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

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
set terminal png size 1200,600
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

// createLineChart takes two datasets (data1 and data2) as slices of float64,
// reduces them to their first 20%, and generates a line chart with two labeled lines.
// The chart is saved to the specified file with dimensions of 10x4 inches.
func createLineChart(data1, data2 []float64, filename string) error {
	n1 := len(data1) / 5
	n2 := len(data2) / 5
	data1 = data1[:n1]
	data2 = data2[:n2]

	// Create a new plot
	p := plot.New()

	// Create line charts for data1 and data2
	line1, err := createLine(data1)
	if err != nil {
		return fmt.Errorf("failed to create line for data1: %v", err)
	}
	line1.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red
	line1.Width = vg.Points(1)                           // Thinner line

	line2, err := createLine(data2)
	if err != nil {
		return fmt.Errorf("failed to create line for data2: %v", err)
	}
	line2.Color = color.RGBA{R: 0, G: 255, B: 0, A: 255} // Green with 50% transparency
	line2.Width = vg.Points(1)                           // Thinner line

	// Add lines to the plot
	p.Add(line1, line2)

	// Add legend
	p.Legend.Add("Original", line1)
	p.Legend.Add("Recreated", line2)

	// Position the legend
	p.Legend.Top = true
	p.Legend.Left = true

	// Hide axes
	p.HideAxes()
	// Save the plot to a file with a larger width
	if err := p.Save(10*vg.Inch, 4*vg.Inch, filename); err != nil {
		return fmt.Errorf("failed to save plot: %v", err)
	}

	return nil
}

// Helper function to create a line chart from []float64
func createLine(data []float64) (*plotter.Line, error) {
	points := make(plotter.XYs, len(data))
	for i, y := range data {
		points[i].X = float64(i) // Use index as X value
		points[i].Y = y
	}
	return plotter.NewLine(points)
}
