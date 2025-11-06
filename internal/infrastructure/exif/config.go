package exif

type ExtensionProviderConfig struct {
	ExifToolCmd      string `yaml:"exiftool_cmd"`
	ExifToolListFlag string `yaml:"exiftool_list_flag"`
}

type RemoverConfig struct {
	Timeout     int    `yaml:"timeout_in_ms"`
	ExifToolCmd string `yaml:"exiftool_cmd"`
}
