package exif

import (
	"os/exec"
	"strings"
	"sync"
)

type ExtensionProvider struct {
	exiftoolCmd         string
	supportedExtensions map[string]bool
	initErr             error
	once                sync.Once
}

// NewExtensionProvider creates a new ExtensionProvider instance that can fetch
// and cache supported file extensions from ExifTool
func NewExtensionProvider(exiftoolCmd string) *ExtensionProvider {
	return &ExtensionProvider{
		exiftoolCmd: exiftoolCmd,
	}
}

// GetSupportedExtensions returns a map of supported file extensions (with dot prefix).
// The extensions are fetched from ExifTool on first call and cached for subsequent calls.
// Returns a copy of the map to prevent external modification.
func (e *ExtensionProvider) GetSupportedExtensions() (map[string]bool, error) {
	e.once.Do(func() {
		e.supportedExtensions, e.initErr = e.fetchSupportedExtensions()
	})

	if e.initErr != nil {
		return nil, e.initErr
	}

	result := make(map[string]bool, len(e.supportedExtensions))
	for k, v := range e.supportedExtensions {
		result[k] = v
	}
	return result, nil
}

// fetchSupportedExtensions executes 'exiftool -listf' command to retrieve all
// supported file extensions. Each extension is stored with a dot prefix and in lowercase.
func (e *ExtensionProvider) fetchSupportedExtensions() (map[string]bool, error) {
	var cmd *exec.Cmd
	if e.exiftoolCmd != "" {
		cmd = exec.Command(e.exiftoolCmd, "-listf")
	} else {
		cmd = exec.Command("exiftool", "-listf")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	extensions := make(map[string]bool)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, ":") {
			continue
		}

		fields := strings.Fields(line)
		for _, ext := range fields {
			if ext != "" {
				extensions["."+strings.ToLower(ext)] = true
			}
		}
	}

	return extensions, nil
}
