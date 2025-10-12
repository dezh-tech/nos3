package exif

// ExtensionProvider provides supported file extensions
type ExtensionProvider interface {
	GetSupportedExtensions() (map[string]bool, error)
}
