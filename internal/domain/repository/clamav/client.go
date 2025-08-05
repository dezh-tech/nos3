package clamav

import (
	"github.com/dutchcoders/go-clamd"
	"io"
)

type ClamdClient interface {
	Ping() error
	ScanStream(reader io.Reader, abort chan bool) (chan *clamd.ScanResult, error)
}
