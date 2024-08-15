package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var store = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

const maxLength = 6

// Load data from the JSON file
func loadDataFromFile() {
	file, err := os.Open("data.json")
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&store.data); err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
}

// Save data to the JSON file
func saveDataToFile() {
	store.RLock()
	defer store.RUnlock()

	file, err := os.Create("data.json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(store.data); err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
}

// Generate a short URL
func generateShortURL(url string) string {
	hash := md5.Sum([]byte(url))
	hashedString := hex.EncodeToString(hash[:])
	short := hashedString[:maxLength]

	store.Lock()
	store.data[short] = url
	store.Unlock()

	saveDataToFile()
	return short
}

// Find the original URL from the short URL
func findOriginalURL(shortURL string) string {
	store.RLock()
	defer store.RUnlock()

	return store.data[shortURL]
}

// Shorten the given URL
func shorten(c *gin.Context) {
	url := c.Param("url")[1:] // Remove the leading '/'
	if len(url) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Please enter the URL"})
		return
	}

	shortenedURL := generateShortURL(url)
	baseURL := os.Getenv("BASE_URL") // Railway will use this environment variable
	short := baseURL + "/" + shortenedURL
	c.IndentedJSON(http.StatusOK, gin.H{"shortened_url": short})
}

// Get all data
func getAllData(c *gin.Context) {
	store.RLock()
	defer store.RUnlock()
	c.IndentedJSON(http.StatusOK, store.data)
}

// Get the original URL from a short URL
func getOriginalURL(c *gin.Context) {
	shortURL := c.Param("url")
	original := findOriginalURL(shortURL)
	if len(original) > 0 {
		c.IndentedJSON(http.StatusOK, gin.H{"original_url": original})
	} else {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Not found"})
	}
}

// Redirect to the original URL
func redirect(c *gin.Context) {
	shortURL := c.Param("shorturl")
	original := findOriginalURL(shortURL)
	if len(original) > 0 {
		c.Redirect(http.StatusFound, original)
	} else {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "URL not found"})
	}
}

// Main function
func main() {
	router := gin.Default()

	// Custom CORS configuration to allow your frontend origin
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://127.0.0.1:5500", "https://shotnerurls.netlify.app/"}
	router.Use(cors.New(corsConfig))

	// Route definitions
	router.GET("/shorten/*url", shorten)
	router.GET("/:shorturl", redirect)
	router.GET("/original/:url", getOriginalURL)

	// Use the PORT environment variable from Railway
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to 8080 if PORT is not set
	}

	router.Run(":" + port) // Start the server on the assigned port
}
