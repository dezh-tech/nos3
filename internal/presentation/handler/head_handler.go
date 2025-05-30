package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"nos3/internal/application/usecase/abstraction"
	"nos3/internal/presentation"

	"github.com/labstack/echo/v4"
)

type HeadHandler struct {
	getter abstraction.Getter
}

func NewHeadHandler(getter abstraction.Getter) *HeadHandler {
	return &HeadHandler{
		getter: getter,
	}
}

// HandleHead handles HEAD /<sha256> requests.
func (h *HeadHandler) HandleHead(c echo.Context) error {
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

	c.Response().Header().Set("Accept-Ranges", "bytes")
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", blob.Size))
	c.Response().Header().Set("Content-Type", blob.BlobType)

	return c.NoContent(http.StatusOK)
}

func removeFileExtension(sha256 string) string {
	if dotIndex := strings.LastIndex(sha256, "."); dotIndex != -1 && len(sha256)-dotIndex <= 5 {
		potentialHash := sha256[:dotIndex]
		if validateSHA256(potentialHash) {
			sha256 = potentialHash
		}
	}

	return sha256
}

func validateSHA256(sha256 string) bool {
	sha256Regex := regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

	return sha256Regex.MatchString(sha256)
}
