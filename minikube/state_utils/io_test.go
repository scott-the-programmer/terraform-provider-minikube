package state_utils

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestReadContents(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write some content to the temporary file
	testContent := "This is a test content."
	_, err = tempFile.WriteString(testContent)
	if err != nil {
		t.Fatalf("Error writing to temp file: %v", err)
	}

	// Close the temporary file
	tempFile.Close()

	// Test the ReadContents function
	result, err := ReadContents(tempFile.Name())
	if err != nil {
		t.Fatalf("Error reading contents: %v", err)
	}

	// Check if the content read matches the expected content
	if result != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, result)
	}
}
