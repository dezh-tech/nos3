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
	contentLength := c.Request().Header.Get("Content-Length")
	contentSize, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		logger.Info("could not parse content length", "error", err)
		contentSize = -1
	}

	hash, _ := c.Get("t").(string)
	author, _ := c.Get("pk").(string)
	result, err := h.uploader.Upload(context.Background(), body, contentSize, hash, contentType, author)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, dto.BlobDescriptor{
		URL:      result.Location,
		Sha256:   hash,
		Size:     result.Size,
		FileType: result.Type,
		Uploaded: time.Now().Unix(),
	})
}
