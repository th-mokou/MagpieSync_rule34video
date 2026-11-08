//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"

	syncpkg "magpiesync/internal/sync"
)

var (
	profileNameRe = regexp.MustCompile(`class="ma-me__name">([^<]+)<`)
	memberIDRe    = regexp.MustCompile(`class="ma-me__link"[^>]*href="[^"]*/members/(\d+)/"`)
	csrfTokenRe   = regexp.MustCompile(`name="remember_me_csrf_token" value="([^"]+)"`)
	pagerPagesRe  = regexp.MustCompile(`data-rd-jump-pages="(\d+)"`)
	pagerPageRe   = regexp.MustCompile(`data-rd-jump-page="(\d+)"`)
)

type Playlist struct {
	ID    string
	Title string
	Count int
}

func parseProfile(body []byte) (Profile, error) {
	s := string(body)
	name := profileNameRe.FindStringSubmatch(s)
	uid := memberIDRe.FindStringSubmatch(s)
	if len(name) == 0 && len(uid) == 0 {
		return Profile{}, ErrAuth
	}
	p := Profile{}
	if len(uid) > 1 {
		p.UID = uid[1]
	}
	if len(name) > 1 {
		p.Name = strings.TrimSpace(html.UnescapeString(name[1]))
	}
	if p.UID == "" {
		return Profile{}, ErrAuth
	}
	if p.Name == "" {
		p.Name = p.UID
	}
	return p, nil
}

func csrfToken(body []byte) string {
	m := csrfTokenRe.FindSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return html.UnescapeString(string(m[1]))
}

func parsePager(body []byte) (page, pages int) {
	if m := pagerPageRe.FindSubmatch(body); len(m) > 1 {
		page, _ = strconv.Atoi(string(m[1]))
	}
	if m := pagerPagesRe.FindSubmatch(body); len(m) > 1 {
		pages, _ = strconv.Atoi(string(m[1]))
	}
	return page, pages
}

func folderPath(remoteID string, page int) (string, error) {
	if page < 1 {
		page = 1
	}
	var path string
	switch {
	case remoteID == folderFavorites:
		path = "my/favourites/videos/"
	case remoteID == folderWatchLater:
		path = "my/favourites/videos-watch-later/"
	case strings.HasPrefix(remoteID, playlistPrefix):
		id := strings.TrimPrefix(remoteID, playlistPrefix)
		if !digits(id) {
			return "", fmt.Errorf("rule34video: unsupported folder %q", remoteID)
		}
		path = "my/playlists/" + id + "/"
	default:
		return "", fmt.Errorf("rule34video: unsupported folder %q", remoteID)
	}
	return fmt.Sprintf("%s?from_my_fav_videos=%d", path, page), nil
}

func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseVideoCards(body []byte) []syncpkg.RemoteItem {
	z := xhtml.NewTokenizer(bytes.NewReader(body))
	var out []syncpkg.RemoteItem
	current := -1
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			return out
		}
		if tt != xhtml.StartTagToken && tt != xhtml.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		if !hasAttr {
			continue
		}
		attrs := tagAttrs(z)
		switch string(name) {
		case "div":
			classes := classTokens(attrs["class"])
			if containsClass(classes, "item") && (containsClass(classes, "ma-v") || containsClass(classes, "ma-prow")) {
				id := strings.TrimSpace(attrs["data-video-card-id"])
				if id == "" || !digits(id) {
					current = -1
					continue
				}
				url := resolveURL(attrs["data-url"])
				if url == "" {
					url = BaseURL + "/video/" + id + "/"
				}
				duration, _ := strconv.Atoi(attrs["data-dur"])
				out = append(out, syncpkg.RemoteItem{
					RemoteID:   id,
					Kind:       2,
					Title:      strings.TrimSpace(html.UnescapeString(attrs["data-title"])),
					URL:        url,
					Duration:   duration,
					AuthorName: "",
				})
				current = len(out) - 1
			}
		case "img":
			if current < 0 || out[current].CoverURL != "" {
				continue
			}
			// The live site uses a data:image placeholder in src and the real
			// thumbnail in data-rd-src, so prefer the lazy source and reject
			// data URIs instead of storing a 1x1 transparent GIF.
			src := strings.TrimSpace(attrs["data-rd-src"])
			if src == "" || isDataURI(src) {
				src = strings.TrimSpace(attrs["src"])
			}
			if src == "" || isDataURI(src) {
				continue
			}
			out[current].CoverURL = resolveURL(src)
		}
	}
}

func parsePlaylists(body []byte) []Playlist {
	z := xhtml.NewTokenizer(bytes.NewReader(body))
	var out []Playlist
	seen := map[string]bool{}
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			return out
		}
		if tt != xhtml.StartTagToken && tt != xhtml.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		if !hasAttr || string(name) != "div" {
			continue
		}
		attrs := tagAttrs(z)
		if !containsClass(classTokens(attrs["class"]), "ma-pl") {
			continue
		}
		id := strings.TrimSpace(attrs["data-id"])
		if id == "" || !digits(id) || seen[id] {
			continue
		}
		seen[id] = true
		count, _ := strconv.Atoi(attrs["data-count"])
		title := strings.TrimSpace(html.UnescapeString(attrs["data-title"]))
		if title == "" {
			title = id
		}
		out = append(out, Playlist{ID: id, Title: title, Count: count})
	}
}

func tagAttrs(z *xhtml.Tokenizer) map[string]string {
	attrs := map[string]string{}
	for {
		key, val, more := z.TagAttr()
		if string(key) == "" {
			break
		}
		attrs[strings.ToLower(string(key))] = string(val)
		if !more {
			break
		}
	}
	return attrs
}

func classTokens(s string) []string {
	return strings.Fields(s)
}

func containsClass(classes []string, want string) bool {
	for _, c := range classes {
		if c == want {
			return true
		}
	}
	return false
}

func resolveURL(raw string) string {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
		return ""
	case strings.HasPrefix(raw, "//"):
		return "https:" + raw
	case strings.HasPrefix(raw, "/"):
		return BaseURL + raw
	case strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "https://"):
		return raw
	default:
		return BaseURL + "/" + strings.TrimLeft(raw, "/")
	}
}

func isDataURI(raw string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "data:")
}

var downloadQualityRe = regexp.MustCompile(`_(\d{3,4})(?:p)?\.mp4`)
var downloadTextQualityRe = regexp.MustCompile(`(\d{3,4})p`)

// parseDownloadOptions extracts the official download anchors from a video
// page. The site renders them as a.tag_item_download with a /get_file/ href
// carrying download=true; the quality is present in the filename.
func parseDownloadOptions(body []byte) []DownloadOption {
	z := xhtml.NewTokenizer(bytes.NewReader(body))
	seen := map[int]bool{}
	var out []DownloadOption
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			break
		}
		if tt != xhtml.StartTagToken && tt != xhtml.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		if !hasAttr || string(name) != "a" {
			continue
		}
		attrs := tagAttrs(z)
		href := html.UnescapeString(strings.TrimSpace(attrs["href"]))
		if href == "" || !strings.Contains(href, "/get_file/") || !strings.Contains(href, "download=true") {
			continue
		}
		if !containsClass(classTokens(attrs["class"]), "tag_item_download") {
			continue
		}
		quality := 0
		if m := downloadQualityRe.FindStringSubmatch(href); len(m) > 1 {
			quality, _ = strconv.Atoi(m[1])
		}
		if quality <= 0 {
			if m := downloadTextQualityRe.FindStringSubmatch(href); len(m) > 1 {
				quality, _ = strconv.Atoi(m[1])
			}
		}
		if quality <= 0 || seen[quality] {
			continue
		}
		seen[quality] = true
		out = append(out, DownloadOption{
			Quality: quality,
			Label:   fmt.Sprintf("%dP", quality),
			URL:     resolveURL(href),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Quality > out[j].Quality })
	return out
}
