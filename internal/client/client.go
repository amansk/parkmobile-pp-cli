package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amansk/parkmobile-pp-cli/internal/auth"
	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
)

// Client calls ParkMobile Phonixx consumer session APIs.
type Client struct {
	BaseURL   string
	HTTP      *http.Client
	Session   *auth.Session
	UserAgent string
	DryRun    bool
	// SourceAppKey is sent when set (mobile app header; unverified without live session).
	SourceAppKey string
}

func New(session *auth.Session) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Session: session,
		UserAgent: "parkmobile-pp-cli/0.1.0 (+https://github.com/amansk/parkmobile-pp-cli)",
	}
}

func (c *Client) url(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + path
}

func (c *Client) applyAuth(req *http.Request) {
	if c.Session == nil {
		return
	}
	if ch := c.Session.CookieHeader(); ch != "" {
		req.Header.Set("Cookie", ch)
	}
	if pm := c.Session.PMAuthenticationToken(); pm != "" {
		req.Header.Set("PMAuthenticationToken", pm)
	}
	if bearer := c.Session.AuthBearer(); bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	} else if pm := c.Session.PMAuthenticationToken(); pm != "" {
		// unverified: mobile clients may also send Bearer; try PMA token as Bearer fallback.
		req.Header.Set("Authorization", "Bearer "+pm)
	}
	if c.SourceAppKey != "" {
		req.Header.Set("SourceAppKey", c.SourceAppKey)
	}
}

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.url(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Origin", "https://parkmobile.io")
	req.Header.Set("Referer", "https://parkmobile.io/")
	c.applyAuth(req)
	return req, nil
}

func (c *Client) doRaw(method, path string, body any) ([]byte, int, error) {
	if c.DryRun && method != http.MethodGet && method != http.MethodHead {
		return nil, 0, exitcode.Usagef("dry-run: would %s %s", method, path)
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := c.newRequest(method, path, rdr)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, exitcode.Transientf("request failed: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return raw, resp.StatusCode, mapHTTPStatus(resp.StatusCode, raw)
}

func mapHTTPStatus(status int, raw []byte) error {
	if status == http.StatusTooManyRequests {
		return exitcode.Transientf("rate limited (HTTP 429)")
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return exitcode.Authf("HTTP %d: %s", status, truncate(string(raw), 200))
	}
	if status == http.StatusNotFound {
		return exitcode.NotFoundf("HTTP %d: %s", status, truncate(string(raw), 200))
	}
	if status >= 400 {
		return exitcode.APIf("HTTP %d: %s", status, truncate(string(raw), 200))
	}
	return nil
}

func (c *Client) doJSON(method, path string, body any, out any) error {
	raw, status, err := c.doRaw(method, path, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if len(raw) == 0 {
		if status >= 400 {
			return exitcode.APIf("HTTP %d empty body", status)
		}
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return exitcode.APIf("decode response: %v (body: %s)", err, truncate(string(raw), 200))
	}
	return nil
}

func (c *Client) doJSONMap(method, path string, body any) (map[string]any, error) {
	raw, _, err := c.doRaw(method, path, body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, exitcode.APIf("decode response: %v", err)
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Ping checks API reachability without auth.
func (c *Client) Ping() error {
	var out struct {
		Locations []any `json:"locations"`
	}
	if err := c.doJSON(http.MethodGet, PathLocations, nil, &out); err != nil {
		return err
	}
	if len(out.Locations) == 0 {
		return exitcode.APIf("locations returned empty")
	}
	return nil
}

// ProbeSessionAuth verifies cookie session using account identify (read-only).
func (c *Client) ProbeSessionAuth() error {
	_, err := c.GetAccount()
	return err
}

// DryRunRequest returns what would be sent for a mutation.
func (c *Client) DryRunRequest(method, path string, body any) map[string]any {
	return map[string]any{
		"method": method,
		"path":   path,
		"url":    c.url(path),
		"body":   body,
	}
}

// APIErrorDetail extracts responseStatus from ServiceStack errors when present.
func APIErrorDetail(raw []byte) string {
	var env struct {
		ResponseStatus struct {
			Message string `json:"message"`
			ErrorCode string `json:"errorCode"`
		} `json:"responseStatus"`
	}
	if err := json.Unmarshal(raw, &env); err == nil && env.ResponseStatus.Message != "" {
		return fmt.Sprintf("%s (%s)", env.ResponseStatus.Message, env.ResponseStatus.ErrorCode)
	}
	return truncate(string(raw), 200)
}
