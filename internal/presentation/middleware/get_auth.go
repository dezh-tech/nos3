package middleware

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"

	"nos3/internal/presentation"
)

func AuthGetMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			authHeader := ctx.Request().Header.Get(presentation.AuthKey)
			url := ctx.Scheme() + "://" + ctx.Request().Host
			providedHash := ctx.Param(presentation.Sha256Param)

			event, err := decodeEvent(authHeader)
			if err != nil {
				ctx.Response().Header().Set("X-Reason", err.Error())

				return ctx.NoContent(http.StatusUnauthorized)
			}

			if err := validateGetEvent(event, providedHash, url); err != nil {
				ctx.Response().Header().Set("X-Reason", err.Error())

				return ctx.NoContent(http.StatusUnauthorized)
			}

			return next(ctx)
		}
	}
}

func validateGetEvent(event *nostr.Event, providedHash, url string) error {
	xTag := getTagValue(event, presentation.XTag)
	serverTag := getTagValue(event, presentation.ServerTag)

	if xTag != providedHash && serverTag != url {
		return errors.New("invalid `x` and `server` tag")
	}

	if !validateSHA256(providedHash) {
		return errors.New("invalid SHA256 hash")
	}

	return nil
}

func validateSHA256(sha256 string) bool {
	sha256 = removeFileExtension(sha256)
	sha256Regex := regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

	return sha256Regex.MatchString(sha256)
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
