package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"nos3/internal/presentation"

	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/assert"
)

const SecretKey = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		setupRequest    func() *http.Request
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "Missing Authorization header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "missing Authorization header",
		},
		{
			name: "Wrong prefix",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Bearer sometoken")

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "missing Nostr header prefix",
		},
		{
			name: "Invalid base64 event",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr invalid-base64")

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "decode base64 event failed",
		},
		{
			name: "Invalid JSON",
			setupRequest: func() *http.Request {
				badJSON := base64.StdEncoding.EncodeToString([]byte(`not a json`))
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr "+badJSON)

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

				eventBytes, err := json.Marshal(invalidEvent)
				if err != nil {
					panic(err)
				}

				encoded := base64.StdEncoding.EncodeToString(eventBytes)
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr "+encoded)

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "invalid signature",
		},
		{
			name: "Invalid kind",
			setupRequest: func() *http.Request {
				event := generateSignedEvent(t, 9999, "upload", 600)
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr "+event)

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "invalid kind",
		},
		{
			name: "CreatedAt in the future",
			setupRequest: func() *http.Request {
				event := generateSignedEventWithCreatedAt(t, 24242, "upload", 600, nostr.Timestamp(time.Now().Unix()+1000))
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr "+event)

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "invalid created_at",
		},
		{
			name: "Missing expiration tag",
			setupRequest: func() *http.Request {
				event := generateSignedEventWithoutTags(t, 24242)
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr "+event)

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "empty expiration tag",
		},
		{
			name: "Expiration expired",
			setupRequest: func() *http.Request {
				event := generateSignedEvent(t, 24242, "upload", -10)
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr "+event)

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "invalid expiration",
		},
		{
			name: "Wrong action",
			setupRequest: func() *http.Request {
				event := generateSignedEvent(t, 24242, "download", 600)
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Authorization", "Nostr "+event)

				return req
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "invalid action",
		},
		{
			name: "Success",
			setupRequest: func() *http.Request {
				event := generateSignedEventWithCorrectX(t, 24242, "upload", 600, strings.NewReader("Hello World!"))
				req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("Hello World!"))
				req.Header.Set("Authorization", "Nostr "+event)

				return req
			},
			expectedStatus:  http.StatusOK,
			expectedMessage: "success",
		},
	}

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

func generateSignedEvent(t *testing.T, kind int, action string, expirationOffset int64) string {
	t.Helper()
	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
		Tags: nostr.Tags{
			{"expiration", strconv.FormatInt(time.Now().Unix()+expirationOffset, 10)},
			{presentation.KeyTraceID, action},
			{"x", "file_id"},
		},
		Content: "",
	}

	_ = event.Sign(SecretKey)
	eventBytes, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(eventBytes)
}

func generateSignedEventWithCorrectX(t *testing.T, kind int, action string, expirationOffset int64, reader io.Reader) string {
	t.Helper()

	bodyContent, _ := io.ReadAll(reader)
	hash := sha256.New()
	hash.Write(bodyContent)
	hashedData := hash.Sum(nil)
	hexHash := hex.EncodeToString(hashedData)

	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
		Tags: nostr.Tags{
			{"expiration", strconv.FormatInt(time.Now().Unix()+expirationOffset, 10)},
			{presentation.KeyTraceID, action},
			{"x", hexHash},
		},
		Content: "",
	}

	_ = event.Sign(SecretKey)
	eventBytes, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(eventBytes)
}

func generateSignedEventWithCreatedAt(t *testing.T, kind int, action string, expirationOffset int64, timestamp nostr.Timestamp) string {
	t.Helper()
	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: timestamp,
		Tags: nostr.Tags{
			{"expiration", strconv.FormatInt(time.Now().Unix()+expirationOffset, 10)},
			{presentation.KeyTraceID, action},
			{"x", "file_id"},
		},
		Content: "",
	}

	_ = event.Sign(SecretKey)
	eventBytes, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(eventBytes)
}

func generateSignedEventWithoutTags(t *testing.T, kind int) string {
	t.Helper()
	event := &nostr.Event{
		Kind:      kind,
		CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
		Tags:      nostr.Tags{},
		Content:   "",
	}
	_ = event.Sign(SecretKey)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(eventBytes)
}
