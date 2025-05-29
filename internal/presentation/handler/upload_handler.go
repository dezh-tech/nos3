package handler

import (
	"net/http"
	"time"

	"nos3/internal/presentation"

	"github.com/labstack/echo/v4"

	"nos3/internal/application/usecase/abstraction"
	"nos3/internal/domain/dto"
)

type UploadHandler struct {
	uploader abstraction.Uploader
}

func NewUploadHandler(uploader abstraction.Uploader) *UploadHandler {
	return &UploadHandler{
		uploader: uploader,
	}
}
func (h *UploadHandler) Handle(c echo.Context) error {
	body := c.Request().Body
	contentType := c.Request().Header.Get(presentation.TypeKey)
	contentSize := c.Request().ContentLength

	hash := c.Get(presentation.XTag).(string)
	author := c.Get(presentation.PK).(string)

	result, err := h.uploader.Upload(c.Request().Context(), body, contentSize, hash, contentType, author)
	if err != nil {
		return c.JSON(result.Status, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, dto.BlobDescriptor{
		URL:      result.Location,
		Sha256:   hash,
		Size:     result.Size,
		FileType: result.Type,
		Uploaded: time.Now().Unix(),
	})
}
