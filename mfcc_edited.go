package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"
)

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
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid STFT value: %v", err)
			}
			row[i] = val
		}
		stft = append(stft, row)
	}
	return stft, scanner.Err()
}

// Mel filterbank generation
func generateMelFilterbank(numFilters, fftSize, sampleRate int) [][]float64 {
	fMin := 0.0
	fMax := float64(sampleRate) / 2.0
	melMin := 2595 * math.Log10(1+fMin/700.0)
	melMax := 2595 * math.Log10(1+fMax/700.0)
	melPoints := make([]float64, numFilters+2)

	for i := range melPoints {
		melPoints[i] = melMin + float64(i)*(melMax-melMin)/float64(numFilters+1)
	}

	hzPoints := make([]float64, len(melPoints))
	for i, mel := range melPoints {
		hzPoints[i] = 700 * (math.Pow(10, mel/2595.0) - 1)
	}

	bins := make([]int, len(hzPoints))
	for i, hz := range hzPoints {
		bins[i] = int(math.Floor((fftSize + 1) * hz / float64(sampleRate)))
	}

	filters := make([][]float64, numFilters)
	for i := range filters {
		filters[i] = make([]float64, fftSize/2+1)
		for j := bins[i]; j < bins[i+1]; j++ {
			filters[i][j] = (float64(j) - float64(bins[i])) / (float64(bins[i+1]) - float64(bins[i]))
		}
		for j := bins[i+1]; j < bins[i+2]; j++ {
			filters[i][j] = (float64(bins[i+2]) - float64(j)) / (float64(bins[i+2]) - float64(bins[i+1]))
		}
	}
	return filters
}

// Function to apply Mel filterbank
func applyMelFilterbank(stft [][]float64, filterbank [][]float64) [][]float64 {
	numFrames := len(stft)
	numFilters := len(filterbank)
	melSpectrogram := make([][]float64, numFrames)

	for t := range stft {
		melSpectrogram[t] = make([]float64, numFilters)
		for m := range filterbank {
			sum := 0.0
			for k, weight := range filterbank[m] {
				sum += stft[t][k] * weight
			}
			melSpectrogram[t][m] = sum
		}
	}
	return melSpectrogram
}

// Logarithm of Mel spectrogram - original looks good
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

// Function to compute MFCCs using custom DCT - original looks good
func computeMFCC(logMelSpectrogram [][]float64, numCoefficients int) [][]float64 {
	mfcc := make([][]float64, len(logMelSpectrogram))
	for i := range logMelSpectrogram {
		mfcc[i] = dctTransform(logMelSpectrogram[i], numCoefficients)
	}
	return mfcc
}

// Function to save MFCC as an image with proper scaling
func saveMFCCImage(mfcc [][]float64, filename string) error {
	height := len(mfcc)
	width := len(mfcc[0])

	img := image.NewGray(image.Rect(0, 0, width, height))

	// Find min and max values of MFCC to normalize the range
	var minVal, maxVal float64
	minVal = math.Inf(1)
	maxVal = math.Inf(-1)

	// Finding the min and max values of the MFCC for scaling
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

	// Scaling MFCC values to [0, 255]
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			scaledValue := (mfcc[y][x] - minVal) / (maxVal - minVal) * 255
			img.SetGray(x, y, color.Gray{Y: uint8(scaledValue)})
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func main() {
	stft, err := parseSTFT("STFT.txt")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	numMelFilters := 40
	numMFCC := 13
	sampleRate := 16000 // Example sample rate
	fftSize := len(stft[0]) * 2

	filterbank := generateMelFilterbank(numMelFilters, fftSize, sampleRate)
	melSpectrogram := applyMelFilterbank(stft, filterbank)
	logMel := logMelSpectrogram(melSpectrogram)
	mfcc := computeMFCC(logMel, numMFCC)

	if err := saveMFCCImage(mfcc, "MFCC.png"); err != nil {
		fmt.Println("Error saving MFCC image:", err)
	} else {
		fmt.Println("MFCC image saved successfully!")
	}
}
