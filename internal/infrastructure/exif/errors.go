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

type ExifError struct {
	Code    ErrorCode
	Message string
	Details string
}

func (e *ExifError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}

	return e.Message
}

func (e *ExifError) IsTimeout() bool {
	return e.Code == ErrorCodeTimeout
}

func (e *ExifError) IsUnsupportedFileType() bool {
	return e.Code == ErrorCodeUnsupportedFileType
}

func (e *ExifError) IsTempFileError() bool {
	return e.Code == ErrorCodeTempFileCreationFailed
}
