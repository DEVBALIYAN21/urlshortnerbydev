package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

var store = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

const maxLength = 6
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

func findOriginalURL(shortURL string) string {
	store.RLock()
	defer store.RUnlock()

	return store.data[shortURL]
}

func shorten(c *gin.Context) {
	url := c.Param("url")[1:]
	if len(url) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Please enter the URL"})
		return
	}

	shortenedURL := generateShortURL(url)
	short := "http://localhost:8080/" + shortenedURL
	c.IndentedJSON(http.StatusOK, gin.H{"shortened_url": short})
}

func getAllData(c *gin.Context) {
	store.RLock()
	defer store.RUnlock()
	c.IndentedJSON(http.StatusOK, store.data)
}

func getOriginalURL(c *gin.Context) {
	shortURL := c.Param("url")
	original := findOriginalURL(shortURL)
	if len(original) > 0 {
		c.IndentedJSON(http.StatusOK, gin.H{"original_url": original})
	} else {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Not found"})
	}
}
func redirect(c *gin.Context) {
	shortURL := c.Param("shorturl")
	if shortURL[0] == '/' {
		shortURL = shortURL[1:]
	}
	original := findOriginalURL(shortURL)
	if len(original) > 0 {
		c.Redirect(http.StatusFound, original)
	} else {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "URL not found"})
	}
}

func main() {
	loadDataFromFile()

	router := gin.Default()
	router.GET("/shorten/*url", shorten)
	router.GET("/shorten", getAllData)
	router.GET("/:shorturl", redirect)
	router.GET("/original/:url", getOriginalURL)
	router.Run("localhost:8080")
}
