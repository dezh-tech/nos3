package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"
)

func authMiddleware(action string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			authHeader := ctx.Request().Header.Get("Authorization")
			if err := validateAuthHeader(authHeader); err != nil {
				return ctx.String(http.StatusUnauthorized, err.Error())
			}

			event, err := decodeEvent(authHeader)
			if err != nil {
				return ctx.String(http.StatusUnauthorized, err.Error())
			}

			if err := validateEvent(event, action); err != nil {
				return ctx.String(http.StatusUnauthorized, err.Error())
			}

			if err := validateRequestBody(event, action, ctx.Request().Body); err != nil {
				return ctx.String(http.StatusUnauthorized, err.Error())
			}

			ctx.Set("pk", event.PubKey)
			ctx.Set("x", getTagValue(event, "x"))
			ctx.Set("t", getTagValue(event, "t"))
			ctx.Set("expiration", getExpirationTime(event))

			return next(ctx)
		}
	}
}

func validateRequestBody(event *nostr.Event, action string, body io.ReadCloser) error {
	defer body.Close()
	if action == "upload" {
		bodyContent, err := io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("could not read request body: %w", err)
		}

		hash := sha256.New()
		hash.Write(bodyContent)
		hashedData := hash.Sum(nil)
		hexHash := hex.EncodeToString(hashedData)
		x := getTagValue(event, "x")

		if hexHash != x {
			return fmt.Errorf("invalid sha256 hash as `x` tag")
		}

		return nil
	}

	return nil
}

func validateAuthHeader(authHeader string) error {
	if authHeader == "" {
		return fmt.Errorf("missing Authorization header")
	}
	if !strings.HasPrefix(authHeader, "Nostr ") {
		return fmt.Errorf("missing Nostr header prefix")
	}

	return nil
}

func decodeEvent(authHeader string) (*nostr.Event, error) {
	eventBase64 := strings.TrimPrefix(authHeader, "Nostr ")
	eventBytes, err := base64.StdEncoding.DecodeString(eventBase64)
	if err != nil {
		return nil, fmt.Errorf("decode base64 event failed: %s", err.Error())
	}

	event := &nostr.Event{}
	if err = json.Unmarshal(eventBytes, event); err != nil {
		return nil, fmt.Errorf("json decode failed: %s", err.Error())
	}

	return event, nil
}

func validateEvent(event *nostr.Event, action string) error {
	if ok, err := event.CheckSignature(); !ok || err != nil {
		return fmt.Errorf("invalid signature")
	}
	if event.Kind != 24242 {
		return fmt.Errorf("invalid kind")
	}
	if event.CreatedAt.Time().Unix() >= time.Now().Unix() {
		return fmt.Errorf("invalid created_at")
	}

	expiration := getTagValue(event, "expiration")
	if expiration == "" {
		return fmt.Errorf("empty expiration tag")
	}

	t := getTagValue(event, "t")
	if t == "" {
		return fmt.Errorf("empty t tag")
	}
	if t != action {
		return fmt.Errorf("invalid action")
	}

	x := getTagValue(event, "x")
	if action == "delete" && x == "" {
		return fmt.Errorf("%s requires `x` tag", action)
	}

	expirationTime, err := strconv.Atoi(expiration)
	if err != nil || time.Unix(int64(expirationTime), 0).Unix() < time.Now().Unix() {
		return fmt.Errorf("invalid expiration")
	}

	return nil
}

func getTagValue(event *nostr.Event, tagName string) string {
	for _, tag := range event.Tags {
		if len(tag) == 2 && tag[0] == tagName {
			return tag[1]
		}
	}

	return ""
}

func getExpirationTime(event *nostr.Event) int {
	expiration := getTagValue(event, "expiration")
	expirationTime, _ := strconv.Atoi(expiration)

	return expirationTime
}
