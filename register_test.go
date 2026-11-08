//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"testing"

	"magpiesync/internal/sites"
)

func TestRule34VideoSiteDescriptor(t *testing.T) {
	site, ok := sites.Get("rule34video")
	if !ok {
		t.Fatal("rule34video site is not registered")
	}
	if !site.Ready || !site.Download {
		t.Fatalf("rule34video readiness/download = %+v", site)
	}
	if len(site.DownloadQualities) != 4 {
		t.Fatalf("rule34video download qualities = %+v", site.DownloadQualities)
	}
	if len(site.Login.Methods) == 0 || site.Login.Methods[0].ID != "password" {
		t.Fatalf("rule34video login methods = %+v", site.Login.Methods)
	}
	if asset, ok := sites.Asset("rule34video", "logo.png"); !ok || len(asset) == 0 {
		t.Fatal("rule34video logo is not registered")
	}
	want := map[string]bool{"favorite": true, "watch_later": true, "playlist": true}
	for _, c := range site.Categories {
		delete(want, c.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing categories: %v", want)
	}
	if _, ok := sites.DriverFor("rule34video"); !ok {
		t.Fatal("rule34video driver is not registered")
	}
}
