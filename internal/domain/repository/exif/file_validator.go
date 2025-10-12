package exif

// FileValidator validates file types
type FileValidator interface {
	ValidateFileType(filePath string) error
}
