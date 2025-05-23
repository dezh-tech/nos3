package entity

type MinIOUploadResult struct {
	Size       int64
	Type       string
	Location   string
	Bucket     string
	HTTPStatus int
}
