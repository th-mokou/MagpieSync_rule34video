//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"testing"
)

func TestParseDownloadOptions(t *testing.T) {
	body := []byte(`<html><body>
		<a class="tag_item tag_item_download vp-sheet__item" href="https://rule34video.com/get_file/51/abc/4447000/4447513/4447513_720p.mp4/?v-acctoken=x&amp;download=true&amp;download_filename=clip_720p.mp4"><svg></svg>MP4 720p<span>193 MB</span></a>
		<a class="tag_item tag_item_download vp-sheet__item" href="https://rule34video.com/get_file/51/def/4447000/4447513/4447513_480p.mp4/?v-acctoken=y&amp;download=true"><span>MP4 480p</span></a>
		<a class="tag_item tag_item_download vp-sheet__item" href="https://rule34video.com/get_file/51/ghi/4447000/4447513/4447513_360.mp4/?v-acctoken=z&amp;download=true"><span>MP4 360p</span></a>
		<a class="video-link" href="https://rule34video.com/get_file/51/abc/4447000/4447513/4447513.mp4/">stream</a>
	</body></html>`)
	opts := parseDownloadOptions(body)
	if len(opts) != 3 {
		t.Fatalf("options = %+v", opts)
	}
	if opts[0].Quality != 720 || opts[1].Quality != 480 || opts[2].Quality != 360 {
		t.Fatalf("qualities = %+v", opts)
	}
	if opts[0].Label != "720P" || opts[0].URL == "" {
		t.Fatalf("first option = %+v", opts[0])
	}
}

func TestParseDownloadOptionsRejectsNonDownloadLinks(t *testing.T) {
	body := []byte(`<a class="tag_item_download" href="https://rule34video.com/get_file/1/a/1/1_720p.mp4/">no download flag</a>`)
	if opts := parseDownloadOptions(body); len(opts) != 0 {
		t.Fatalf("options = %+v", opts)
	}
}
