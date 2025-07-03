package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"nos3/internal/application/usecase/abstraction"

	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"

	"nos3/internal/presentation"
	"nos3/pkg/logger"
)

type ReportHandler struct {
	reporter abstraction.Reporter
}

func NewReportHandler(reporter abstraction.Reporter) *ReportHandler {
	return &ReportHandler{
		reporter: reporter,
	}
}

// HandleReport handles PUT /report requests.
// The request body MUST be a signed NIP-56 report event.
func (h *ReportHandler) HandleReport(c echo.Context) error {
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		c.Response().Header().Set(presentation.ReasonTag, "failed to read request body")

		return c.NoContent(http.StatusBadRequest)
	}

	event := &nostr.Event{}
	if err := json.Unmarshal(bodyBytes, event); err != nil {
		c.Response().Header().Set(presentation.ReasonTag, "invalid JSON in request body: "+err.Error())

		return c.NoContent(http.StatusBadRequest)
	}

	if err := h.validateNIP56ReportEvent(event); err != nil {
		c.Response().Header().Set(presentation.ReasonTag, err.Error())

		return c.NoContent(http.StatusUnauthorized)
	}

	pubKey := event.PubKey
	var blobHashes []string
	var reportType string
	var eventID string
	var serverURL string

	for _, tag := range event.Tags {
		if len(tag) >= 2 {
			switch tag[0] {
			case presentation.XTag:
				if len(tag) >= 3 {
					blobHashes = append(blobHashes, tag[1])
					reportType = tag[2]
				}
			case presentation.ETag:
				if len(tag) >= 2 {
					eventID = tag[1]
				}
			case presentation.ServerTag:
				if len(tag) >= 2 {
					serverURL = tag[1]
				}
			}
		}
	}

	if len(blobHashes) == 0 || reportType == "" {
		c.Response().Header().Set(presentation.ReasonTag, "missing required 'x' tags or report type in Nostr event")

		return c.NoContent(http.StatusBadRequest)
	}

	err = h.reporter.ReportBlob(c.Request().Context(), pubKey, blobHashes, reportType, eventID, event.Content, serverURL)
	if err != nil {
		logger.Error("failed to process blob report", "err", err)
		c.Response().Header().Set(presentation.ReasonTag, err.Error())

		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

// validateNIP56ReportEvent performs NIP-56 specific validations on the Nostr event.
func (h *ReportHandler) validateNIP56ReportEvent(event *nostr.Event) error {
	if ok, err := event.CheckSignature(); !ok || err != nil {
		return errors.New("invalid signature")
	}

	if event.Kind != 1984 {
		return errors.New("invalid kind, expected 1984 for report event")
	}

	if event.CreatedAt.Time().Unix() > time.Now().Add(1*time.Minute).Unix() {
		return errors.New("invalid created_at: event timestamp is in the future")
	}

	pTag := event.Tags.Find(presentation.PTag)
	if len(pTag) < 2 || pTag[1] == "" {
		return errors.New("missing or empty 'p' tag (reported pubkey) in Nostr event")
	}

	expiration := h.getTagValue(event, presentation.ExpTag)
	if expiration != "" {
		expirationTime, err := strconv.Atoi(expiration)
		if err != nil || time.Unix(int64(expirationTime), 0).Unix() < time.Now().Unix() {
			return errors.New("invalid expiration tag")
		}
	}

	hasValidXTag := false
	for _, tag := range event.Tags {
		if len(tag) >= 3 && tag[0] == presentation.XTag {
			if !h.validateSHA256(tag[1]) {
				return errors.New("invalid SHA256 hash in 'x' tag")
			}

			switch tag[2] {
			case presentation.NudityReport, presentation.ImpersonationReport,
				presentation.IllegalReport, presentation.MalwareReport, presentation.ProfanityReport,
				presentation.OtherReport, presentation.SpamReport:
				hasValidXTag = true
			default:
				return errors.New("invalid report type in 'x' tag")
			}
		}
	}

	if !hasValidXTag {
		return errors.New("missing or invalid 'x' tag with a valid report type")
	}

	return nil
}

func (h *ReportHandler) getTagValue(event *nostr.Event, tagName string) string {
	tag := event.Tags.Find(tagName)
	if len(tag) > 1 {
		return tag[1]
	}

	return ""
}

func (h *ReportHandler) validateSHA256(sha256 string) bool {
	return sha256Regex.MatchString(sha256)
}
