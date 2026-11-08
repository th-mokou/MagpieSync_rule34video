//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParseCredentialRequiresSession(t *testing.T) {
	if _, err := ParseCredential([]byte(`{"cookies":{"__ddg1_":"x"}}`)); err == nil {
		t.Fatal("credential without PHPSESSID accepted")
	}
	raw, _ := json.Marshal(Credential{Cookies: map[string]string{"PHPSESSID": "session"}})
	cred, err := ParseCredential(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Cookies["PHPSESSID"] != "session" {
		t.Fatalf("credential = %+v", cred)
	}
}

func TestProfileRejectsAnonymousPage(t *testing.T) {
	if _, err := parseProfile([]byte(`<html><body><form action="/login/">Login</form></body></html>`)); !errors.Is(err, ErrAuth) {
		t.Fatalf("anonymous page error = %v, want ErrAuth", err)
	}
}

func TestCSRFToken(t *testing.T) {
	body := []byte(`<form><input type="hidden" name="remember_me_csrf_token" value="abc&amp;def"></form>`)
	if got := csrfToken(body); got != "abc&def" {
		t.Fatalf("csrfToken = %q", got)
	}
}
