package client

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
)

const UnverifiedBodyHint = "mutation request bodies are scaffolded from public ServiceStack metadata only (order_token + credit_card); live shapes need HAR verification — use --dry-run or pass --acknowledge-unverified-body after review"

// StartPreviewInput for session start preview (quote only).
type StartPreviewInput struct {
	ZoneCode        string
	DurationMinutes int
	VehicleID       int
	BillingMethodID int
	SpaceNumber     string
	OrderToken      string
	TimeBlockID     int
}

// StartPreviewResult is a quote-only preview (no charge).
type StartPreviewResult struct {
	ZoneCode         string  `json:"zone_code"`
	DurationMinutes  int     `json:"duration_minutes"`
	TotalPrice       float64 `json:"total_price"`
	ParkingPrice     float64 `json:"parking_price"`
	ServiceFee       float64 `json:"service_fee"`
	IsParkingAllowed bool    `json:"is_parking_allowed"`
	NotAllowedReason string  `json:"not_allowed_reason,omitempty"`
	DryRun           bool    `json:"dry_run"`
	Message          string  `json:"message"`
	UnverifiedNote   string  `json:"unverified_note,omitempty"`
}

// StartSessionInput for live session start.
type StartSessionInput struct {
	OrderToken      string
	BillingMethodID int
}

// ExtendSessionInput for live session extend.
type ExtendSessionInput struct {
	OrderToken      string
	BillingMethodID int
}

// StopSessionInput for live session stop.
type StopSessionInput struct {
	SessionID int
}

func (c *Client) guardLiveMutation() error {
	if c.DryRun {
		return nil
	}
	if !c.AllowUnverifiedMutations {
		return exitcode.Usagef("refusing live mutation: %s", UnverifiedBodyHint)
	}
	return nil
}

func activateBody(orderToken string, billingMethodID int) map[string]any {
	body := map[string]any{}
	if orderToken != "" {
		body["order_token"] = orderToken
	}
	if billingMethodID > 0 {
		body["credit_card"] = map[string]any{
			"billing_method_id": billingMethodID,
		}
	}
	return body
}

// StartPreview returns a price quote without starting parking.
// GET /v3/parking/price expects order_token + duration_in_minutes per metadata; zone-only quotes are unverified.
func (c *Client) StartPreview(in StartPreviewInput) (StartPreviewResult, error) {
	result := StartPreviewResult{
		ZoneCode:        in.ZoneCode,
		DurationMinutes: in.DurationMinutes,
		DryRun:          true,
		Message:         "Preview only — no charge. Use session start with all safety gates to commit.",
		UnverifiedNote:  "GET /v3/parking/price requires order_token per metadata; zone-only preview is unverified without checkout HAR — see PLAN.md",
	}
	if in.OrderToken == "" {
		result.NotAllowedReason = "order_token required for price quote per ServiceStack metadata; capture from browser checkout HAR"
		return result, nil
	}
	quote, err := c.GetPriceQuote(in.OrderToken, in.DurationMinutes, in.TimeBlockID)
	if err != nil {
		return StartPreviewResult{}, err
	}
	result.TotalPrice = quote.TotalPrice
	result.ParkingPrice = quote.ParkingPrice
	result.ServiceFee = quote.ServiceFee
	result.IsParkingAllowed = quote.IsParkingAllowed
	result.NotAllowedReason = quote.NotAllowedReason
	return result, nil
}

// StartSession starts zone parking (hard-gated at CLI layer).
// Metadata shape: POST /v3/parking/active with order_token + credit_card (not zone/duration fields).
func (c *Client) StartSession(in StartSessionInput) (map[string]any, error) {
	body := activateBody(in.OrderToken, in.BillingMethodID)
	if err := c.guardLiveMutation(); err != nil {
		return nil, err
	}
	if c.DryRun {
		return map[string]any{
			"dry_run":    true,
			"would_post": PathActiveV3,
			"request":    c.DryRunRequest(http.MethodPost, PathActiveV3, body),
			"unverified": true,
			"note":       UnverifiedBodyHint,
		}, nil
	}
	return c.doJSONMap(http.MethodPost, PathActiveV3, body)
}

// ExtendSession extends an active parking session.
// Metadata shape: PUT /v3/extension/active with order_token + credit_card (not parkingActionId).
func (c *Client) ExtendSession(in ExtendSessionInput) (map[string]any, error) {
	body := activateBody(in.OrderToken, in.BillingMethodID)
	if err := c.guardLiveMutation(); err != nil {
		return nil, err
	}
	if c.DryRun {
		return map[string]any{
			"dry_run":    true,
			"would_put":  PathExtensionV3,
			"request":    c.DryRunRequest(http.MethodPut, PathExtensionV3, body),
			"unverified": true,
			"note":       UnverifiedBodyHint,
		}, nil
	}
	return c.doJSONMap(http.MethodPut, PathExtensionV3, body)
}

// StopSession stops an active session early when zone allows (CanStop).
func (c *Client) StopSession(in StopSessionInput) (map[string]any, error) {
	if in.SessionID <= 0 {
		return nil, exitcode.Usagef("session id required")
	}
	path := fmt.Sprintf(PathActiveStop, url.PathEscape(fmt.Sprint(in.SessionID)))
	if err := c.guardLiveMutation(); err != nil {
		return nil, err
	}
	if c.DryRun {
		return map[string]any{
			"dry_run":      true,
			"would_delete": path,
			"request":      c.DryRunRequest(http.MethodDelete, path, nil),
			"unverified":   true,
			"note":         "DELETE path verified in metadata; optional JSON body unverified",
		}, nil
	}
	return c.doJSONMap(http.MethodDelete, path, nil)
}
