package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"nos3/internal/application/usecase/abstraction"
	"nos3/internal/domain/dto"
	"nos3/pkg/logger"
)

type UploadHandler struct {
	uploader abstraction.Uploader
}

func (h *UploadHandler) Handle(c echo.Context) error {
	body := c.Request().Body
	contentType := c.Request().Header.Get("Content-Type")
	contentSize := c.Request().ContentLength

	hash, _ := c.Get(KeyTraceID).(string)
	author, _ := c.Get("pk").(string)
	result, err := h.uploader.Upload(context.Background(), body, contentSize, hash, contentType, author)
	if err != nil {
		logger.Error("upload failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to upload file. Please try again later.",
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
