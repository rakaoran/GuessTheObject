package game

import (
	"api/internal/shared/logger"
	"bufio"
	"os"
)

var wordsList []string

// loadWordsFromFile reads a file where each line is a word
// and returns them as a slice of strings.
func LoadWords() {
	// Open the file
	filePath := "./words.txt"
	file, err := os.Open(filePath)
	if err != nil {
		logger.Fatalf("could not open file %s: %w", filePath, err)
	}
	// Make sure to close the file when the function exits
	defer file.Close()

	var words []string
	// Create a new scanner for the file
	scanner := bufio.NewScanner(file)

	// Loop over all lines in the file
	for scanner.Scan() {
		// Append the line (which is our word) to the slice
		words = append(words, scanner.Text())
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		logger.Fatalf("error while reading file %s: %w", filePath, err)
	}

	wordsList = words
	logger.Infof("Words count: %v", len(words))
}
