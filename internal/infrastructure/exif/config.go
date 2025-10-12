package exif

type RemoverConfig struct {
	Timeout     int    `yaml:"timeout_in_ms"`
	ExifToolCmd string `yaml:"exiftool_cmd"`
}
