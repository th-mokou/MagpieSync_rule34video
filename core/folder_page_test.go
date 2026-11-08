//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListFolderPageRejectsUnexpectedMarkup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body>temporary error</body></html>`))
	}))
	defer server.Close()

	c := &Client{http: server.Client(), baseURL: server.URL + "/", userAgent: userAgent}
	if _, err := c.ListFolderPage(context.Background(), folderFavorites, 1, 20); err == nil {
		t.Fatal("expected unrecognized video list page error")
	}
}

func TestListFolderPageAcceptsRecognizedEmptyMarkup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<div id="list_videos_my_favourite_videos_items" data-rdm-items></div>`))
	}))
	defer server.Close()

	c := &Client{http: server.Client(), baseURL: server.URL + "/", userAgent: userAgent}
	page, err := c.ListFolderPage(context.Background(), folderFavorites, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 || page.HasMore {
		t.Fatalf("page = %+v", page)
	}
}
