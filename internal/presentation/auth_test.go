package presentation

import (
	"encoding/base64"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var tests = []struct {
	name            string
	setupRequest    func() *http.Request
	expectedStatus  int
	expectedMessage string
}{
	{
		name: "Missing Authorization header",
		setupRequest: func() *http.Request {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "missing Authorization header",
	},
	{
		name: "Wrong prefix",
		setupRequest: func() *http.Request {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer sometoken")
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "missing Nostr header prefix",
	},
	{
		name: "Invalid base64 event",
		setupRequest: func() *http.Request {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr invalid-base64")
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "decode base64 event failed",
	},
	{
		name: "Invalid JSON",
		setupRequest: func() *http.Request {
			badJson := base64.StdEncoding.EncodeToString([]byte(`not a json`))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+badJson)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "json decode failed",
	},
	{
		name: "Invalid signature",
		setupRequest: func() *http.Request {
			invalidEvent := nostr.Event{
				PubKey:    "invalid",
				Kind:      24242,
				CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
			}

			eventBytes, _ := json.Marshal(invalidEvent)
			encoded := base64.StdEncoding.EncodeToString(eventBytes)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+encoded)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "invalid signature",
	},
	{
		name: "Invalid kind",
		setupRequest: func() *http.Request {
			event := generateSignedEvent(9999, "upload", 600)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+event)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "invalid kind",
	},
	{
		name: "CreatedAt in the future",
		setupRequest: func() *http.Request {
			event := generateSignedEventWithCreatedAt(24242, "upload", 600, nostr.Timestamp(time.Now().Unix()+1000))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+event)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "invalid created_at",
	},
	{
		name: "Missing expiration tag",
		setupRequest: func() *http.Request {
			event := generateSignedEventWithoutTags(24242)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+event)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "empty expiration tag",
	},
	{
		name: "Expiration expired",
		setupRequest: func() *http.Request {
			event := generateSignedEvent(24242, "upload", -10)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+event)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "invalid expiration",
	},
	{
		name: "Wrong action",
		setupRequest: func() *http.Request {
			event := generateSignedEvent(24242, "download", 600)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+event)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "invalid action",
	},
	{
		name: "Upload missing x tag",
		setupRequest: func() *http.Request {
			event := generateSignedEventWithoutX(24242, "upload", 600)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+event)
			return req
		},
		expectedStatus:  http.StatusUnauthorized,
		expectedMessage: "upload requires `x` tag",
	},
	{
		name: "Success",
		setupRequest: func() *http.Request {
			event := generateSignedEvent(24242, "upload", 600)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Nostr "+event)
			return req
		},
		expectedStatus:  http.StatusOK,
		expectedMessage: "success",
	},
}

const SecretKey = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()
	e := echo.New()
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := tt.setupRequest()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mw := authMiddleware("upload")(handler)
			_ = mw(c)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.expectedMessage)
		})
	}
}

func generateSignedEvent(kind int, action string, expirationOffset int64) string {

	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
		Tags: nostr.Tags{
			{"expiration", strconv.FormatInt(time.Now().Unix()+expirationOffset, 10)},
			{"t", action},
			{"x", "file_id"},
		},
		Content: "",
	}

	event.Sign(SecretKey)
	eventBytes, _ := json.Marshal(event)
	return base64.StdEncoding.EncodeToString(eventBytes)
}

func generateSignedEventWithCreatedAt(kind int, action string, expirationOffset int64, timestamp nostr.Timestamp) string {

	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: timestamp,
		Tags: nostr.Tags{
			{"expiration", strconv.FormatInt(time.Now().Unix()+expirationOffset, 10)},
			{"t", action},
			{"x", "file_id"},
		},
		Content: "",
	}

	event.Sign(SecretKey)
	eventBytes, _ := json.Marshal(event)
	return base64.StdEncoding.EncodeToString(eventBytes)
}

func generateSignedEventWithoutTags(kind int) string {
	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
		Tags:      nostr.Tags{},
		Content:   "",
	}
	event.Sign(SecretKey)

	eventBytes, _ := json.Marshal(event)
	return base64.StdEncoding.EncodeToString(eventBytes)
}

func generateSignedEventWithoutX(kind int, action string, expirationOffset int64) string {
	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
		Tags: nostr.Tags{
			{"expiration", strconv.FormatInt(time.Now().Unix()+expirationOffset, 10)},
			{"t", action},
		},
		Content: "",
	}

	event.Sign(SecretKey)
	eventBytes, _ := json.Marshal(event)
	return base64.StdEncoding.EncodeToString(eventBytes)
}
