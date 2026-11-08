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

func TestListFolderPageAcceptsEmptyWatchLaterList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<div class="ma-block" id="list_videos_my_favourite_videos" data-rdm-list="favourites" data-rdm-total="0" data-rdm-pg="1" data-rdm-per="12" data-rdm-of="0"><nav id="list_videos_my_favourite_videos_fav_selector"></nav><form><div class="fs-empty ma-empty"><h3>Nothing to watch later yet</h3></div></form></div>`))
	}))
	defer server.Close()

	c := &Client{http: server.Client(), baseURL: server.URL + "/", userAgent: userAgent}
	page, err := c.ListFolderPage(context.Background(), folderWatchLater, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 || page.HasMore {
		t.Fatalf("page = %+v", page)
	}
}
