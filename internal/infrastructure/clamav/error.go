package clamav

import "fmt"

type ErrorCode int

const (
	ErrorCodeUnknown ErrorCode = iota
	ErrorCodeConnectionFailed
	ErrorCodeScanFailed
	ErrorCodeTimeout
	ErrorCodeMalwareDetected
	ErrorCodeInvalidInput
)

type MalwareError struct {
	Code    ErrorCode
	Message string
	Details string
}

func (e *MalwareError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

func (e *MalwareError) IsTimeout() bool {
	return e.Code == ErrorCodeTimeout
}

func (e *MalwareError) IsMalwareDetected() bool {
	return e.Code == ErrorCodeMalwareDetected
}

func (e *MalwareError) IsConnectionError() bool {
	return e.Code == ErrorCodeConnectionFailed
}
