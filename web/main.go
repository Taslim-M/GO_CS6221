package main

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Serve index.html at the default route
	router.GET("/", func(c *gin.Context) {
		c.File("./index.html")
	})

	// Handle POST request to process file and return base64 image
	router.POST("/", func(c *gin.Context) {
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.String(http.StatusBadRequest, "File upload error")
			return
		}
		defer file.Close()

		// Simulate processing and return a sample base64-encoded image for demonstration
		// Replace this logic with actual MFCC processing and image generation
		dummyImage, _ := ioutil.ReadFile("dummy.jpg")
		base64Image := base64.StdEncoding.EncodeToString(dummyImage)

		c.String(http.StatusOK, base64Image)
	})

	router.Run(":8080") // Start server on port 8080
}
