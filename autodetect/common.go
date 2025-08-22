package autodetect

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteOrAppendToFile writes the content to the specified file.
// If the file exists, it appends the content. If the directory doesn't exist, it creates it.
func WriteOrAppendToFile(filePath, content string) error {
	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755) // Create the directory if it doesn't exist
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", dir, err)
		}
		fmt.Printf("directory %s created\n", dir)
	}

	// Open the file for appending, or create it if it doesn't exist
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("error opening file %s: %s\n", filePath, err.Error())
		return fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer f.Close()

	// Write the content to the file
	_, err = f.WriteString(content)
	if err != nil {
		fmt.Printf("error writing to file %s: %s\n", filePath, err.Error())
		return fmt.Errorf("error writing to file %s: %w", filePath, err)
	}

	return nil
}
