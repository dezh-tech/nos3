package middleware

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"

	"nos3/internal/presentation"
)

func AuthDeleteMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			authHeader := ctx.Request().Header.Get(presentation.AuthKey)
			event, err := decodeEvent(authHeader)
			if err != nil {
				ctx.Response().Header().Set(presentation.ReasonTag, err.Error())

				return ctx.NoContent(http.StatusUnauthorized)
			}

			if err := validateDeleteEvent(event, ctx); err != nil {
				ctx.Response().Header().Set(presentation.ReasonTag, err.Error())

				return ctx.NoContent(http.StatusUnauthorized)
			}

			return next(ctx)
		}
	}
}

func validateDeleteEvent(event *nostr.Event, c echo.Context) error {
	xTag := getTagValue(event, presentation.XTag)
	if xTag == "" {
		return errors.New("missing 'x' tag for delete action")
	}

	paramSha256 := c.Param(presentation.Sha256Param)
	paramSha256 = removeFileExtension(paramSha256)

	if xTag != paramSha256 {
		return errors.New("x tag mismatch with URL sha256 for delete action")
	}

	return nil
}
