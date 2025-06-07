package handler

import (
	"fmt"
	"net/http"

	"nos3/internal/application/usecase/abstraction"
	"nos3/internal/presentation"

	"github.com/labstack/echo/v4"
)

type GetHandler struct {
	getter abstraction.Getter
}

func NewGetHandler(getter abstraction.Getter) *GetHandler {
	return &GetHandler{
		getter: getter,
	}
}

// HandleGet handles GET /<sha256> requests.
func (h *GetHandler) HandleGet(c echo.Context) error {
	sha256 := c.Param(presentation.Sha256Param)
	if sha256 == "" {
		c.Response().Header().Set(presentation.ReasonTag, "missing sha256 hash")

		return c.NoContent(http.StatusBadRequest)
	}

	sha256 = removeFileExtension(sha256)

	blob, err := h.getter.GetBlob(c.Request().Context(), sha256)
	if err != nil {
		c.Response().Header().Set(presentation.ReasonTag, err.Error())

		return c.NoContent(http.StatusNotFound)
	}

	redirectURL := blob.MinIOAddress

	c.Response().Header().Set("Content-Type", blob.BlobType)
	c.Response().Header().Set("Accept-Ranges", "bytes")
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", blob.Size))

	return c.Redirect(http.StatusFound, redirectURL)
}
