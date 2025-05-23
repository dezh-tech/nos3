package dto

type BlobDescriptor struct {
	URL      string `json:"url"`
	Sha256   string `json:"sha256"`
	Size     int64  `json:"size"`
	FileType string `json:"type"`
	Uploaded int64  `json:"uploaded"`
}
