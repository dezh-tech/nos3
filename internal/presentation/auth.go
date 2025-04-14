package presentation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func authMiddleware(action string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {

			authHeader := ctx.Request().Header.Get("Authorization")
			if authHeader == "" {
				return ctx.String(http.StatusUnauthorized, "missing Authorization header")
			}
			if !strings.HasPrefix(authHeader, "Nostr ") {
				return ctx.String(http.StatusUnauthorized, "missing Nostr header prefix")
			}

			eventBase64 := strings.TrimPrefix(authHeader, "Nostr ")
			eventBytes, err := base64.StdEncoding.DecodeString(eventBase64)
			if err != nil {
				return ctx.String(http.StatusUnauthorized,
					fmt.Sprintf("decode base64 event failed : %s", err.Error()))
			}

			event := &nostr.Event{}
			if err = json.Unmarshal(eventBytes, event); err != nil {
				return ctx.String(http.StatusUnauthorized,
					fmt.Sprintf("json decode failed : %s", err.Error()))
			}

			if ok, err := event.CheckSignature(); !ok || err != nil {
				return ctx.String(http.StatusUnauthorized, "invalid signature")
			}

			if event.Kind != 24242 {
				return ctx.String(http.StatusUnauthorized, "invalid kind")
			}

			if event.CreatedAt.Time().Unix() >= time.Now().Unix() {
				return ctx.String(http.StatusUnauthorized, "invalid created_at")
			}

			expiration, t, x := getValues(event)
			if expiration == "" {
				return ctx.String(http.StatusUnauthorized, "empty expiration tag")
			}
			if t == "" {
				return ctx.String(http.StatusUnauthorized, "empty t tag")
			}

			expirationTime, err := strconv.Atoi(expiration)
			if err != nil || time.Unix(int64(expirationTime), 0).Unix() < time.Now().Unix() {
				return ctx.String(http.StatusUnauthorized, "invalid expiration")
			}

			if t != action {
				return ctx.String(http.StatusUnauthorized, "invalid action")
			}

			if (action == "upload" || action == "delete") && x == "" {
				return ctx.String(http.StatusUnauthorized, fmt.Sprintf("%s requires `x` tag", action))
			}

			ctx.Set("pk", event.PubKey)
			ctx.Set("x", x)
			ctx.Set("t", t)
			ctx.Set("expiration", expirationTime)

			return next(ctx)
		}
	}
}

func getValues(event *nostr.Event) (string, string, string) {
	expiration := ""
	t := ""
	x := ""
	for _, tag := range event.Tags {
		if len(tag) == 2 {
			switch tag[0] {
			case "expiration":
				expiration = tag[1]
			case "t":
				t = tag[1]
			case "x":
				x = tag[1]
			}
		}
	}
	return expiration, t, x
}
