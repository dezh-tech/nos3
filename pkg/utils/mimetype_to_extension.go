package utils

import "strings"

// mimeTypeToExtension maps common MIME types to their typical file extensions.
var mimeTypeToExtension = map[string]string{
	"application/json": ".json",
	"application/pdf":  ".pdf",
	"application/xml":  ".xml",
	"application/zip":  ".zip",
	"application/gzip": ".gz",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   ".docx",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
	"application/x-tar":        ".tar",
	"application/vnd.rar":      ".rar",
	"application/x-sh":         ".sh",
	"application/octet-stream": ".bin",
	"audio/aac":                ".aac",
	"audio/mpeg":               ".mp3",
	"audio/ogg":                ".ogg",
	"audio/wav":                ".wav",
	"audio/webm":               ".webm",
	"image/bmp":                ".bmp",
	"image/gif":                ".gif",
	"image/jpeg":               ".jpg",
	"image/png":                ".png",
	"image/svg+xml":            ".svg",
	"image/tiff":               ".tif",
	"image/webp":               ".webp",
	"text/css":                 ".css",
	"text/csv":                 ".csv",
	"text/html":                ".html",
	"text/javascript":          ".js",
	"text/plain":               ".txt",
	"text/xml":                 ".xml",
	"video/avi":                ".avi",
	"video/mpeg":               ".mpeg",
	"video/mp4":                ".mp4",
	"video/ogg":                ".ogv",
	"video/webm":               ".webm",
	"video/x-flv":              ".flv",
	"video/x-ms-wmv":           ".wmv",
}

// GetExtensionFromMimeType returns a common file extension for a given MIME type.
// If no specific extension is found, it defaults to ".bin".
func GetExtensionFromMimeType(mimeType string) string {
	// Remove charset if present (e.g., "text/plain; charset=utf-8")
	cleanedMimeType := strings.Split(mimeType, ";")[0]
	if ext, ok := mimeTypeToExtension[cleanedMimeType]; ok {
		return ext
	}

	return ".bin"
}
