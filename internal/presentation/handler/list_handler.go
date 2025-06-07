package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"nos3/internal/application/usecase/abstraction"
	"nos3/internal/presentation"

	"github.com/labstack/echo/v4"
)

type ListHandler struct {
	lister abstraction.Lister
}

func NewListHandler(lister abstraction.Lister) *ListHandler {
	return &ListHandler{
		lister: lister,
	}
}

// HandleList handles GET /list/:pubKey requests.
func (h *ListHandler) HandleList(c echo.Context) error {
	pubKey := c.Param(presentation.PK)
	if pubKey == "" {
		c.Response().Header().Set(presentation.ReasonTag, "missing pubKey")

		return c.NoContent(http.StatusBadRequest)
	}

	since, err := parseTimeQueryParam(c, "since")
	if err != nil {
		c.Response().Header().Set(presentation.ReasonTag, err.Error())

		return c.NoContent(http.StatusBadRequest)
	}

	until, err := parseTimeQueryParam(c, "until")
	if err != nil {
		c.Response().Header().Set(presentation.ReasonTag, err.Error())

		return c.NoContent(http.StatusBadRequest)
	}

	blobs, status, err := h.lister.ListBlobs(c.Request().Context(), pubKey, since, until)
	if err != nil {
		c.Response().Header().Set(presentation.ReasonTag, err.Error())

		return c.NoContent(status)
	}

	return c.JSON(http.StatusOK, blobs)
}

// parseTimeQueryParam parses a Unix timestamp string from query parameters into a *time.Time.
func parseTimeQueryParam(c echo.Context, paramName string) (*time.Time, error) {
	s := c.QueryParam(paramName)
	if s == "" {
		return nil, nil //nolint
	}

	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid '%s' timestamp", paramName)
	}

	t := time.Unix(ts, 0)

	return &t, nil
}
