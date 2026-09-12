package cli_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/amansk/parkmobile-pp-cli/internal/auth"
	"github.com/amansk/parkmobile-pp-cli/internal/cli"
	"github.com/amansk/parkmobile-pp-cli/internal/client"
)

func mockHTTP(t *testing.T) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/locations":
			_, _ = w.Write([]byte(`{"locations":[{"code":6,"systemName":"United States"}]}`))
		case "/account/identify2":
			_, _ = w.Write([]byte(`{"firstName":"Test","lastName":"User","email":"user@example.com","mobile":"+14155550100"}`))
		case "/account/vehicles":
			_, _ = w.Write([]byte(`{"vehicles":[{"vehicleId":1,"vrn":"ABC123","state":"CA","default":true}]}`))
		case "/account/paymentmethods":
			_, _ = w.Write([]byte(`{"paymentMethods":[{"billingMethodId":10,"last4":"4242","cardType":"Visa","isDefault":true}]}`))
		case "/v4/parking/zone/1234":
			_, _ = w.Write([]byte(`{"signageCode":"1234","locationName":"Test Zone","parkInfo":{"isParkingAllowed":true,"maxParkingTime":{"totalMinutes":120},"timeBlocks":[{"name":"2 Hours","timeBlockUnit":"Minutes","minimumValue":30,"maximumValue":120}]}}`))
		case "/v3/parking/price":
			if r.URL.Query().Get("order_token") == "tok123" {
				_, _ = w.Write([]byte(`{"price":{"totalPrice":5.25,"parkingPrice":4.00,"serviceFee":1.25},"isParkingAllowed":true}`))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
		case "/v2/parking/history":
			_, _ = w.Write([]byte(`{"parkingActions":[{"id":99,"canStop":true,"canExtend":true,"zone":{"signageCode":"1234"},"car":{"vrn":"ABC123","state":"CA"},"priceDetail":{"totalPrice":5.25}}]}`))
		case "/v3/parking/active":
			_, _ = w.Write([]byte(`{"parkingActionId":100,"status":"active"}`))
		case "/v3/extension/active":
			_, _ = w.Write([]byte(`{"parkingActionId":99,"extended":true}`))
		case "/parking/active/99":
			if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte(`{"stopped":true}`))
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	c := client.New(&auth.Session{RawCookieHeader: "PMAuthenticationToken=test"})
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()
	return c
}

func runCLIWithHome(t *testing.T, home string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := cli.NewRootForTest(&cli.Options{
		HTTP: mockHTTP(t),
		Home: home,
	})
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	allArgs := append([]string{"--home", home}, args...)
	root.SetArgs(allArgs)
	code := 0
	if err := root.Execute(); err != nil {
		code = cli.ExitCodeForTest(err)
	}
	return code, stdout.String(), stderr.String()
}

func runCLI(t *testing.T, args ...string) (int, string, string) {
	return runCLIWithHome(t, t.TempDir(), args...)
}

func TestVersion(t *testing.T) {
	code, out, _ := runCLI(t, "--version")
	if code != 0 || out == "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestDoctorJSON(t *testing.T) {
	home := t.TempDir()
	sess := &auth.Session{Cookies: map[string]string{"PMAuthenticationToken": "x"}, Source: "test"}
	if err := auth.SaveSession(home, sess); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLIWithHome(t, home, "--json", "doctor", "--live")
	if code != 0 {
		t.Fatalf("code=%d err=%q out=%q", code, errOut, out)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if report["ok"] != true {
		t.Fatalf("report=%v", report)
	}
}

func TestSessionStartPreviewWithoutOrderToken(t *testing.T) {
	home := t.TempDir()
	sess := &auth.Session{Cookies: map[string]string{"PMAuthenticationToken": "x"}}
	if err := auth.SaveSession(home, sess); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLIWithHome(t, home, "--json", "session", "start", "preview", "--zone", "1234", "--duration-minutes", "60")
	if code != 0 {
		t.Fatalf("code=%d err=%q out=%q", code, errOut, out)
	}
	var preview struct {
		DryRun           bool   `json:"dry_run"`
		NotAllowedReason string `json:"not_allowed_reason"`
	}
	if err := json.Unmarshal([]byte(out), &preview); err != nil {
		t.Fatal(err)
	}
	if !preview.DryRun || preview.NotAllowedReason == "" {
		t.Fatalf("%+v", preview)
	}
}

func TestSessionStartPreviewWithOrderToken(t *testing.T) {
	home := t.TempDir()
	sess := &auth.Session{Cookies: map[string]string{"PMAuthenticationToken": "x"}}
	if err := auth.SaveSession(home, sess); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLIWithHome(t, home, "--json", "session", "start", "preview",
		"--zone", "1234", "--duration-minutes", "60", "--order-token", "tok123")
	if code != 0 {
		t.Fatalf("code=%d err=%q out=%q", code, errOut, out)
	}
	var preview struct {
		DryRun     bool    `json:"dry_run"`
		TotalPrice float64 `json:"total_price"`
	}
	if err := json.Unmarshal([]byte(out), &preview); err != nil {
		t.Fatal(err)
	}
	if !preview.DryRun || preview.TotalPrice != 5.25 {
		t.Fatalf("%+v", preview)
	}
}

func TestSessionStartRefusesWithoutGates(t *testing.T) {
	home := t.TempDir()
	sess := &auth.Session{Cookies: map[string]string{"PMAuthenticationToken": "x"}}
	if err := auth.SaveSession(home, sess); err != nil {
		t.Fatal(err)
	}
	code, _, _ := runCLIWithHome(t, home, "session", "start", "--order-token", "tok123")
	if code != 2 {
		t.Fatalf("code=%d want 2", code)
	}
}

func TestSessionStartDryRunWithGates(t *testing.T) {
	home := t.TempDir()
	sess := &auth.Session{Cookies: map[string]string{"PMAuthenticationToken": "x"}}
	if err := auth.SaveSession(home, sess); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLIWithHome(t, home, "--json", "--dry-run", "session", "start",
		"--order-token", "tok123", "--billing-method-id", "10",
		"--enable-live-parking", "--owner-approved", "--confirm", "START PARKMOBILE SESSION")
	if code != 0 {
		t.Fatalf("code=%d err=%q out=%q", code, errOut, out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["dry_run"] != true {
		t.Fatalf("%v", payload)
	}
	if payload["started"] == true {
		t.Fatalf("dry-run must not report started=true: %v", payload)
	}
}

func TestSessionStartBlocksLiveWithoutAcknowledge(t *testing.T) {
	home := t.TempDir()
	sess := &auth.Session{Cookies: map[string]string{"PMAuthenticationToken": "x"}}
	if err := auth.SaveSession(home, sess); err != nil {
		t.Fatal(err)
	}
	code, _, _ := runCLIWithHome(t, home, "session", "start",
		"--order-token", "tok123", "--billing-method-id", "10",
		"--enable-live-parking", "--owner-approved", "--confirm", "START PARKMOBILE SESSION")
	if code != 2 {
		t.Fatalf("code=%d want 2", code)
	}
}

func TestZonesGetJSON(t *testing.T) {
	home := t.TempDir()
	sess := &auth.Session{Cookies: map[string]string{"PMAuthenticationToken": "x"}}
	if err := auth.SaveSession(home, sess); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLIWithHome(t, home, "--json", "zones", "get", "--zone", "1234", "--duration-minutes", "60")
	if code != 0 {
		t.Fatalf("code=%d err=%q out=%q", code, errOut, out)
	}
}

func TestAuthLoginCookiesFile(t *testing.T) {
	home := t.TempDir()
	cookieFile := home + "/import.txt"
	if err := os.WriteFile(cookieFile, []byte("PMAuthenticationToken=abc123"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLIWithHome(t, home, "--json", "auth", "login", "--cookies-file", cookieFile)
	if code != 0 {
		t.Fatalf("code=%d err=%q out=%q", code, errOut, out)
	}
}
