// Package auth manages ParkMobile consumer session cookies.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppName      = "parkmobile-pp-cli"
	CookiesEnv   = "PARKMOBILE_COOKIES"
	CookieDomain = "parkmobile"
)

// Session holds cookie auth material. Values are never logged.
type Session struct {
	RawCookieHeader string            `json:"cookie_header,omitempty"`
	Cookies         map[string]string `json:"cookies,omitempty"`
	Source          string            `json:"source,omitempty"`
	UpdatedAt       string            `json:"updated_at,omitempty"`
}

// HomeDir resolves the config directory.
func HomeDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if v := os.Getenv("PARKMOBILE_PP_HOME"); v != "" {
		return v, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName), nil
}

func cookiesPath(home string) string {
	return filepath.Join(home, "cookies.json")
}

// LoadSession reads stored cookies (0600 file).
func LoadSession(home string) (*Session, error) {
	path := cookiesPath(home)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse cookies file: %w", err)
	}
	return &s, nil
}

// SaveSession writes cookies with mode 0600.
func SaveSession(home string, s *Session) error {
	if err := os.MkdirAll(home, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := cookiesPath(home)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// CookieValue returns a named cookie from the session, if present.
func (s *Session) CookieValue(name string) string {
	if s == nil {
		return ""
	}
	if s.Cookies != nil {
		if v, ok := s.Cookies[name]; ok {
			return v
		}
	}
	if s.RawCookieHeader == "" {
		return ""
	}
	prefix := name + "="
	for _, part := range strings.Split(s.RawCookieHeader, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, prefix) {
			return part[len(prefix):]
		}
	}
	return ""
}

// PMAuthenticationToken returns the mobile/web auth token cookie when present.
func (s *Session) PMAuthenticationToken() string {
	for _, name := range []string{"PMAuthenticationToken", "pm_auth_token", "pmAuthToken"} {
		if v := s.CookieValue(name); v != "" {
			return v
		}
	}
	return ""
}

// AuthBearer returns a Bearer token from common cookie/header names.
func (s *Session) AuthBearer() string {
	for _, name := range []string{"Authorization", "auth_token", "access_token", "token"} {
		if v := s.CookieValue(name); v != "" {
			v = strings.TrimPrefix(v, "Bearer ")
			return v
		}
	}
	return ""
}

// CookieHeader returns the Cookie header value for HTTP requests.
func (s *Session) CookieHeader() string {
	if s == nil {
		return ""
	}
	if s.RawCookieHeader != "" {
		return s.RawCookieHeader
	}
	if len(s.Cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(s.Cookies))
	for k, v := range s.Cookies {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

// Status returns safe metadata (no secret values).
func (s *Session) Status() map[string]any {
	if s == nil {
		return map[string]any{"authenticated": false}
	}
	names := cookieNames(s)
	hasCookies := s.CookieHeader() != ""
	hasPMToken := s.PMAuthenticationToken() != ""
	hasBearer := s.AuthBearer() != ""
	out := map[string]any{
		"authenticated": hasCookies,
		"has_cookies":   hasCookies,
		"has_pm_token":  hasPMToken,
		"has_bearer":    hasBearer,
		"source":        s.Source,
		"updated_at":    s.UpdatedAt,
		"cookie_names":  names,
		"cookie_count":  len(names),
	}
	if hasCookies && !hasPMToken && !hasBearer {
		out["auth_warning"] = "cookies present but no PMAuthenticationToken/Bearer detected; app.parkmobile.io login alone may not work — complete Zone Parking at dlweb.parkmobile.us/Phonixx/ and re-import"
	}
	return out
}

func cookieNames(s *Session) []string {
	names := make([]string, 0)
	if s.RawCookieHeader != "" {
		for _, part := range strings.Split(s.RawCookieHeader, ";") {
			part = strings.TrimSpace(part)
			if i := strings.Index(part, "="); i > 0 {
				names = append(names, part[:i])
			}
		}
	}
	for k := range s.Cookies {
		names = append(names, k)
	}
	return names
}

// ResolveSession loads file cookies, then PARKMOBILE_COOKIES env.
func ResolveSession(home string) (*Session, error) {
	if v := strings.TrimSpace(os.Getenv(CookiesEnv)); v != "" {
		return ParseCookieInput(v, "env:"+CookiesEnv)
	}
	return LoadSession(home)
}
