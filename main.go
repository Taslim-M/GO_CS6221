package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	filesToDelete := []string{"image.png", "image.txt", "curr.wav", "recreate.wav", "chart.png"}
	// Delete any old  files
	deleteFiles(filesToDelete)

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

	// Endpoint to serve the first audio file
	router.GET("/audio1", func(c *gin.Context) {
		c.File("./curr.wav") // Path to the first audio file
	})

	// Endpoint to serve the second audio file
	router.GET("/audio2", func(c *gin.Context) {
		c.File("./recreate.wav") // Path to the second audio file
	})

	router.GET("/getchart", func(c *gin.Context) {
		c.File("./chart.png") // Path to the chart image file
	})

	router.Run(":8080") // Start server on port 8080
}

// Function to check for errors while reading and writing files
func check(e error) {
	if e != nil {
		panic(e)
	}
}
