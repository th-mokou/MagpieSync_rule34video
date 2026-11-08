//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"testing"
)

func TestParseProfileAndPlaylists(t *testing.T) {
	body := []byte(`<div class="ma-me"><span class="ma-av">KA</span><div class="ma-me__id"><span class="ma-me__name">kamik</span><a class="ma-me__link" href="https://rule34video.com/members/469636/">View public profile</a></div></div>`)
	p, err := parseProfile(body)
	if err != nil {
		t.Fatal(err)
	}
	if p.UID != "469636" || p.Name != "kamik" {
		t.Fatalf("profile = %+v", p)
	}
	playlists := parsePlaylists([]byte(`
		<a class="ma-newcard" href="/create-playlist/">New playlist</a>
		<div class="item ma-pl" data-rdm-item data-id="753975" data-title="233" data-count="2156" data-url="https://rule34video.com/my/playlists/753975/"></div>
		<div class="item ma-pl" data-rdm-item data-id="128913" data-title="Marie &amp; Rose" data-count="6" data-url="/my/playlists/128913/"></div>`))
	if len(playlists) != 2 || playlists[0].ID != "753975" || playlists[0].Count != 2156 {
		t.Fatalf("playlists = %+v", playlists)
	}
	if playlists[1].Title != "Marie & Rose" || playlists[1].Count != 6 {
		t.Fatalf("playlist title = %+v", playlists[1])
	}
}

func TestParseVideoCardsAndPager(t *testing.T) {
	body := []byte(`
		<div id="list_videos_my_favourite_videos_items" data-rdm-items>
			<div class="item ma-v favorite_videos_video_1" data-video-card-id="4447513" data-id="4447513" data-title="Yi Xuan&#039;s Secret" data-dur="612" data-url="https://rule34video.com/video/4447513/yi-xuan/">
				<a class="ma-thumb" href="/video/4447513/yi-xuan/"><img class="thumb" src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" data-rd-src="https://rule34video.com/contents/videos_screenshots/4447000/4447513/336x189/3.jpg"></a>
			</div>
			<div class="item ma-prow favorite_videos_video_2" data-video-card-id="4218826" data-title="1 Hour Of Marie Rose" data-dur="3600" data-url="/video/4218826/one-hour/">
				<img data-rd-src="//rule34video.com/contents/videos_screenshots/4218000/4218826/320x180/1.jpg">
			</div>
		</div>
		<nav class="fs-pager" data-rd-jump-base="?from_my_fav_videos=" data-rd-jump-page="1" data-rd-jump-pages="5"></nav>`)
	items := parseVideoCards(body)
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].RemoteID != "4447513" || items[0].Title != "Yi Xuan's Secret" || items[0].Duration != 612 {
		t.Fatalf("first item = %+v", items[0])
	}
	if items[0].URL != "https://rule34video.com/video/4447513/yi-xuan/" {
		t.Fatalf("first url = %q", items[0].URL)
	}
	if items[0].CoverURL != "https://rule34video.com/contents/videos_screenshots/4447000/4447513/336x189/3.jpg" {
		t.Fatalf("first cover = %q", items[0].CoverURL)
	}
	if items[1].CoverURL != "https://rule34video.com/contents/videos_screenshots/4218000/4218826/320x180/1.jpg" {
		t.Fatalf("second cover = %q", items[1].CoverURL)
	}
	if page, pages := parsePager(body); page != 1 || pages != 5 {
		t.Fatalf("pager = %d/%d", page, pages)
	}
}

func TestFolderPath(t *testing.T) {
	cases := []struct {
		id   string
		page int
		want string
	}{
		{"favorites", 1, "my/favourites/videos/?from_my_fav_videos=1"},
		{"watch_later", 2, "my/favourites/videos-watch-later/?from_my_fav_videos=2"},
		{"playlist:753975", 180, "my/playlists/753975/?from_my_fav_videos=180"},
	}
	for _, tc := range cases {
		got, err := folderPath(tc.id, tc.page)
		if err != nil || got != tc.want {
			t.Fatalf("folderPath(%q,%d) = %q, %v; want %q", tc.id, tc.page, got, err, tc.want)
		}
	}
	if _, err := folderPath("playlist:../../etc", 1); err == nil {
		t.Fatal("invalid playlist id accepted")
	}
}
