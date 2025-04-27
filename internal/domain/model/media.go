package model

import "time"

type Blob struct {
	ID           string      `bson:"_id"`
	MinIOAddress string      `bson:"minio_address"`
	UploadTime   time.Time   `bson:"upload_time"`
	Author       string      `bson:"author"`
	BlobType     string      `bson:"blob_type"`
	Duration     *int        `bson:"duration"`   // Pointer to allow null for non-audio/video
	Dimensions   *Dimensions `bson:"dimensions"` // Pointer to allow null
	Size         int64       `bson:"size"`
	Blurhash     string      `bson:"blurhash"`
	Metadata     []Tag       `bson:"metadata"`
}

type Dimensions struct {
	Width  int `bson:"width"`
	Height int `bson:"height"`
}

type Tag struct {
	Key   string `bson:"key"`
	Value string `bson:"value"`
}
