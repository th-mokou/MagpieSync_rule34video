//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTreatsLoginRedirectAsAuthError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login/?next=/my/", http.StatusFound)
	}))
	defer ts.Close()
	c := &Client{
		http: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
		baseURL: ts.URL + "/",
		cookies: map[string]string{"PHPSESSID": "expired"},
	}
	if _, err := c.get(context.Background(), "my/favourites/videos/"); !errors.Is(err, ErrAuth) {
		t.Fatalf("get error = %v, want ErrAuth", err)
	}
}
