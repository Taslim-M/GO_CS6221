package main

import (
	"fmt"
	"image/color"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func createLineChart(data1, data2 []float64, filename string) error {
	n1 := len(data1) / 10
	n2 := len(data2) / 10
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

	// Save the plot to a file
	if err := p.Save(6*vg.Inch, 4*vg.Inch, filename); err != nil {
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
