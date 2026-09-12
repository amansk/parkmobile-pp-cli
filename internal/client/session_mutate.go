package client

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
)

// StartPreviewInput for session start preview (quote only).
type StartPreviewInput struct {
	ZoneCode        string
	DurationMinutes int
	VehicleID       int
	BillingMethodID int
	SpaceNumber     string
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
	// unverified: full checkout body TBD from live HAR
	UnverifiedNote string `json:"unverified_note,omitempty"`
}

// StartSessionInput for live session start.
type StartSessionInput struct {
	ZoneCode        string
	DurationMinutes int
	VehicleID       int
	BillingMethodID int
	SpaceNumber     string
	OrderToken      string
}

// ExtendSessionInput for live session extend.
type ExtendSessionInput struct {
	SessionID       int
	DurationMinutes int
	BillingMethodID int
	TimeBlockID     int
}

// StopSessionInput for live session stop.
type StopSessionInput struct {
	SessionID int
}

// StartPreview returns a price quote without starting parking.
func (c *Client) StartPreview(in StartPreviewInput) (StartPreviewResult, error) {
	quote, err := c.GetPriceQuote(in.ZoneCode, in.DurationMinutes, "")
	if err != nil {
		return StartPreviewResult{}, err
	}
	return StartPreviewResult{
		ZoneCode:         in.ZoneCode,
		DurationMinutes:  in.DurationMinutes,
		TotalPrice:       quote.TotalPrice,
		ParkingPrice:     quote.ParkingPrice,
		ServiceFee:       quote.ServiceFee,
		IsParkingAllowed: quote.IsParkingAllowed,
		NotAllowedReason: quote.NotAllowedReason,
		DryRun:           true,
		Message:          "Preview only — no charge. Use session start with all safety gates to commit.",
		UnverifiedNote:   "POST /v3/parking/active body shape unverified without live session; see PLAN.md",
	}, nil
}

// StartSession starts zone parking (hard-gated at CLI layer).
// unverified: request body derived from ParkingActivateRequest CSV metadata + Hackatrain 2019 patterns.
func (c *Client) StartSession(in StartSessionInput) (map[string]any, error) {
	body := map[string]any{
		"zoneCode":          in.ZoneCode,
		"durationInMinutes": in.DurationMinutes,
	}
	if in.VehicleID > 0 {
		body["vehicleId"] = in.VehicleID
	}
	if in.BillingMethodID > 0 {
		body["billingMethodId"] = in.BillingMethodID
	}
	if in.SpaceNumber != "" {
		body["spaceNumber"] = in.SpaceNumber
	}
	if in.OrderToken != "" {
		body["orderToken"] = in.OrderToken
	}
	if c.DryRun {
		return map[string]any{
			"dry_run":    true,
			"would_post": PathActiveV3,
			"request":    c.DryRunRequest(http.MethodPost, PathActiveV3, body),
			"unverified": true,
		}, nil
	}
	return c.doJSONMap(http.MethodPost, PathActiveV3, body)
}

// ExtendSession extends an active parking session.
// unverified: PUT /v3/extension/active body shape from ParkingExtensionActivateRequest metadata.
func (c *Client) ExtendSession(in ExtendSessionInput) (map[string]any, error) {
	body := map[string]any{
		"parkingActionId":   in.SessionID,
		"durationInMinutes": in.DurationMinutes,
	}
	if in.BillingMethodID > 0 {
		body["billingMethodId"] = in.BillingMethodID
	}
	if in.TimeBlockID > 0 {
		body["timeblockId"] = in.TimeBlockID
	}
	if c.DryRun {
		return map[string]any{
			"dry_run":    true,
			"would_put":  PathExtensionV3,
			"request":    c.DryRunRequest(http.MethodPut, PathExtensionV3, body),
			"unverified": true,
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
	if c.DryRun {
		return map[string]any{
			"dry_run":      true,
			"would_delete": path,
			"request":      c.DryRunRequest(http.MethodDelete, path, nil),
			"unverified":   true,
		}, nil
	}
	return c.doJSONMap(http.MethodDelete, path, nil)
}
