package clamav

type ScannerConfig struct {
	Address string `yaml:"address"`
	Timeout int    `yaml:"timeout"`
}
