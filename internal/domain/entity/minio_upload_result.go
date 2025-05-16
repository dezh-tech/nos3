package entity

type MinIOUploadResult struct {
	Size     int64  `json:"size"`
	Type     string `json:"type"`
	Location string `json:"location"`
	Bucket   string `json:"bucket"`
}
