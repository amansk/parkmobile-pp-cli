package auth_test

import (
	"testing"

	"github.com/amansk/parkmobile-pp-cli/internal/auth"
)

func TestParseCookieInput(t *testing.T) {
	s, err := auth.ParseCookieInput("foo=bar; PMAuthenticationToken=secret", "test")
	if err != nil {
		t.Fatal(err)
	}
	if s.CookieValue("foo") != "bar" {
		t.Fatalf("foo=%q", s.CookieValue("foo"))
	}
	st := s.Status()
	if st["authenticated"] != true {
		t.Fatalf("status=%v", st)
	}
	if _, ok := st["cookie_names"]; !ok {
		t.Fatal("expected cookie_names")
	}
}

func TestParseCookiesFileStorageState(t *testing.T) {
	data := []byte(`{"cookies":[{"name":"sid","value":"abc","domain":".parkmobile.io"}]}`)
	s, err := auth.ParseCookiesFile(data, "test")
	if err != nil {
		t.Fatal(err)
	}
	if s.CookieValue("sid") != "abc" {
		t.Fatalf("sid=%q", s.CookieValue("sid"))
	}
}

func TestParseCookiesFileRejectsEmpty(t *testing.T) {
	_, err := auth.ParseCookiesFile([]byte("   "), "test")
	if err == nil {
		t.Fatal("expected error")
	}
}
