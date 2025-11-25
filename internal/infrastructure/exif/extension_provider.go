package exif

import (
	"strings"
	"sync"

	"nos3/internal/domain/repository/cliexecuter"
)

type ExtensionProvider struct {
	exiftoolCmd         string
	listFlag            string
	executor            cliexecuter.CommandExecutor
	supportedExtensions map[string]bool
	initErr             error
	once                sync.Once
}

// NewExtensionProvider creates a new ExtensionProvider instance that can fetch
// and cache supported file extensions from ExifTool.
func NewExtensionProvider(cfg ExtensionProviderConfig, executor cliexecuter.CommandExecutor) *ExtensionProvider {
	return &ExtensionProvider{
		exiftoolCmd: cfg.ExifToolCmd,
		listFlag:    cfg.ExifToolListFlag,
		executor:    executor,
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
	output, err := e.executor.Execute(e.exiftoolCmd, e.listFlag)
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
