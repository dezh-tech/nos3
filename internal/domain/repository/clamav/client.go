package clamav

import (
	"io"

	"github.com/dutchcoders/go-clamd"
)

type ClamdClient interface {
	Ping() error
	ScanStream(reader io.Reader, abort chan bool) (chan *clamd.ScanResult, error)
}
