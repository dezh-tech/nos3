package handler

import (
	"net/http"

	"nos3/internal/application/usecase/abstraction"
	"nos3/internal/presentation"

	"github.com/labstack/echo/v4"
)

type DeleteHandler struct {
	deleter abstraction.Deleter
}

func NewDeleteHandler(deleter abstraction.Deleter) *DeleteHandler {
	return &DeleteHandler{
		deleter: deleter,
	}
}

// HandleDelete handles DELETE /:sha256 requests.
func (h *DeleteHandler) HandleDelete(c echo.Context) error {
	sha256 := c.Param(presentation.Sha256Param)
	if sha256 == "" {
		c.Response().Header().Set(presentation.ReasonTag, "missing sha256 hash")

		return c.NoContent(http.StatusBadRequest)
	}

	sha256 = removeFileExtension(sha256)

	status, err := h.deleter.DeleteBlob(c.Request().Context(), sha256)
	if err != nil {
		c.Response().Header().Set(presentation.ReasonTag, err.Error())

		return c.NoContent(status)
	}

	return c.NoContent(http.StatusOK)
}
