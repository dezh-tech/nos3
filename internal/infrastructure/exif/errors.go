package exif

import "fmt"

type ErrorCode int

const (
	ErrorCodeUnknown ErrorCode = iota
	ErrorCodeExifToolNotFound
	ErrorCodeTempFileCreationFailed
	ErrorCodeExifRemovalFailed
	ErrorCodeTimeout
	ErrorCodeInvalidInput
	ErrorCodeUnsupportedFileType
)

type Error struct {
	Code    ErrorCode
	Message string
	Details string
}

func (e *Error) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}

	return e.Message
}

func (e *Error) IsTimeout() bool {
	return e.Code == ErrorCodeTimeout
}

func (e *Error) IsUnsupportedFileType() bool {
	return e.Code == ErrorCodeUnsupportedFileType
}

func (e *Error) IsTempFileError() bool {
	return e.Code == ErrorCodeTempFileCreationFailed
}
