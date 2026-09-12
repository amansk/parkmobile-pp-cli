package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ImportChromeCookies extracts parkmobile cookies from a Chrome profile.
// Sign in at parkmobile.io/login, choose Zone Parking (dlweb.parkmobile.us/Phonixx/), then import.
func ImportChromeCookies() (*Session, error) {
	profileDir, err := defaultChromeProfile()
	if err != nil {
		return nil, err
	}
	cookiesDB := filepath.Join(profileDir, "Cookies")
	if _, err := os.Stat(cookiesDB); err != nil {
		return nil, fmt.Errorf("chrome cookies database not found at %s: sign in at parkmobile.io/login, choose Zone Parking, then retry", cookiesDB)
	}
	if sess, err := importViaPython(cookiesDB); err == nil {
		return sess, nil
	}
	return importViaSQLiteCopy(cookiesDB)
}

func defaultChromeProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default"), nil
	case "linux":
		return filepath.Join(home, ".config", "google-chrome", "Default"), nil
	case "windows":
		return filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default"), nil
	default:
		return "", fmt.Errorf("unsupported OS for --chrome: %s", runtime.GOOS)
	}
}

func importViaPython(cookiesDB string) (*Session, error) {
	script := `
import sys, json
try:
    import browser_cookie3
except ImportError:
    sys.exit(2)
jar = browser_cookie3.chrome(cookie_file=sys.argv[1])
out = {}
for c in jar:
    d = c.domain.lower()
    if "parkmobile" in d or "phonixx" in d:
        out[c.name] = c.value
print(json.dumps(out))
`
	cmd := exec.Command("python3", "-c", script, cookiesDB)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("chrome cookie import (install browser_cookie3: pip install browser_cookie3): %w", err)
	}
	var cookies map[string]string
	if err := json.Unmarshal(out, &cookies); err != nil {
		return nil, err
	}
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no parkmobile/phonixx cookies in chrome; sign in at https://parkmobile.io/login, choose Zone Parking (dlweb.parkmobile.us), then retry")
	}
	return &Session{
		Cookies:   cookies,
		Source:    "chrome",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func importViaSQLiteCopy(cookiesDB string) (*Session, error) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return nil, fmt.Errorf("chrome import needs python3+browser_cookie3 (recommended) or sqlite3; raw sqlite reads encrypted values on modern Chrome — use auth login --cookies-file from a HAR/export instead")
	}
	tmp := filepath.Join(os.TempDir(), "parkmobile-chrome-cookies.db")
	copyCmd := exec.Command("cp", cookiesDB, tmp)
	if err := copyCmd.Run(); err != nil {
		return nil, fmt.Errorf("copy chrome cookies db: %w", err)
	}
	defer os.Remove(tmp)
	query := `SELECT name, value, host_key FROM cookies WHERE host_key LIKE '%parkmobile%' OR host_key LIKE '%phonixx%'`
	cmd := exec.Command("sqlite3", tmp, query)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read chrome cookies: %w", err)
	}
	cookies := map[string]string{}
	hosts := map[string]struct{}{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) == 3 {
			cookies[parts[0]] = parts[1]
			hosts[parts[2]] = struct{}{}
		}
	}
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no parkmobile/phonixx cookies in chrome; sign in at https://parkmobile.io/login, choose Zone Parking (dlweb.parkmobile.us), then retry")
	}
	for _, v := range cookies {
		if strings.HasPrefix(v, "v10") || strings.HasPrefix(v, "v11") {
			return nil, fmt.Errorf("chrome cookie values appear encrypted; install python3+browser_cookie3 (pip install browser_cookie3) or export cookies via --cookies-file")
		}
	}
	sess := &Session{
		Cookies:   cookies,
		Source:    "chrome-sqlite",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if sess.PMAuthenticationToken() == "" {
		return nil, fmt.Errorf("imported %d cookies from chrome but no PMAuthenticationToken; complete Zone Parking login at dlweb.parkmobile.us/Phonixx/ (parkmobile.io alone seeds CSRF cookies only) — hosts seen: %s", len(cookies), hostKeys(hosts))
	}
	return sess, nil
}

func hostKeys(hosts map[string]struct{}) string {
	keys := make([]string, 0, len(hosts))
	for k := range hosts {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
