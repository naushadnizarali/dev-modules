package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadFile downloads a file from a given URL and returns the local path
func DownloadFile(url string) (string, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Extract fileName from URL
	fileName := filepath.Base(resp.Request.URL.Path)

	// Create a temp file with proper extension
	tmpFile, err := os.CreateTemp("", "email-att-*"+filepath.Ext(fileName))
	if err != nil {
		return "", "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", "", err
	}

	return tmpFile.Name(), fileName, nil
}
