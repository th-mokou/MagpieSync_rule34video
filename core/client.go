//go:build !no_rule34video && (!explicit_sites || site_rule34video)

// Package rule34video is a read-only Rule 34 Video adapter. It only reads the
// account's favourites, watch-later list and playlists. The only POST is the
// login form; no write endpoint or CSRF write token is ever used.
package rule34video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"magpiesync/internal/netproxy"
	syncpkg "magpiesync/internal/sync"
)

const (
	BaseURL = "https://rule34video.com"

	requestTimeout = 45 * time.Second
	maxBodyBytes   = 8 << 20
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36"

	folderFavorites  = "favorites"
	folderWatchLater = "watch_later"
	playlistPrefix   = "playlist:"
)

var (
	// ErrAuth means the saved session is missing or has expired.
	ErrAuth = errors.New("rule34video: authentication required")
	// ErrInvalidCredentials means the submitted login form was rejected.
	ErrInvalidCredentials = errors.New("rule34video: invalid username or password")
	// ErrChallenge means DDoS-Guard or a rate limit blocked the request.
	ErrChallenge = errors.New("rule34video: challenge or rate limit")
)

type HTTPError struct {
	StatusCode int
	Path       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("rule34video: HTTP %d for %s", e.StatusCode, e.Path)
}

func IsAuth(err error) bool {
	return errors.Is(err, ErrAuth)
}

func IsFatal(err error) bool {
	return errors.Is(err, ErrAuth) || errors.Is(err, ErrChallenge)
}

// Credential is the encrypted account payload. It contains session cookies
// only and no site write token.
type Credential struct {
	Cookies   map[string]string `json:"cookies"`
	UserAgent string            `json:"user_agent,omitempty"`
}

func ParseCredential(raw []byte) (*Credential, error) {
	var c Credential
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse rule34video credential: %w", err)
	}
	if c.Cookies == nil || (strings.TrimSpace(c.Cookies["PHPSESSID"]) == "" && strings.TrimSpace(c.Cookies["__Host-kt_member_v2"]) == "") {
		return nil, errors.New("rule34video credential is missing PHPSESSID or __Host-kt_member_v2")
	}
	return &c, nil
}

type Profile struct {
	UID       string
	Name      string
	AvatarURL string
}

type Client struct {
	http      *http.Client
	cookies   map[string]string
	baseURL   string
	userAgent string

	mu      sync.Mutex
	profile *Profile
}

func NewClient(cookies map[string]string, proxyURL string) (*Client, error) {
	return NewClientWithUserAgent(cookies, proxyURL, "")
}

func NewClientWithUserAgent(cookies map[string]string, proxyURL, ua string) (*Client, error) {
	hc, err := netproxy.NewClient(proxyURL, requestTimeout)
	if err != nil {
		return nil, err
	}
	// A logged-out session redirects protected pages to the login page. Do not
	// follow that redirect: treat it as an auth error instead of parsing the
	// login page as an empty folder, which could make a short folder look gone.
	hc.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	c := &Client{
		http:      hc,
		cookies:   cloneCookies(cookies),
		baseURL:   BaseURL + "/",
		userAgent: userAgent,
	}
	if strings.TrimSpace(ua) != "" {
		if strings.ContainsAny(ua, "\r\n") {
			hc.CloseIdleConnections()
			return nil, errors.New("rule34video: invalid user agent")
		}
		c.userAgent = ua
	}
	return c, nil
}

func (c *Client) Close() error {
	c.http.CloseIdleConnections()
	return nil
}

func cloneCookies(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (c *Client) Profile(ctx context.Context) (Profile, error) {
	if strings.TrimSpace(c.cookies["PHPSESSID"]) == "" && strings.TrimSpace(c.cookies["__Host-kt_member_v2"]) == "" {
		return Profile{}, ErrAuth
	}
	c.mu.Lock()
	if c.profile != nil {
		p := *c.profile
		c.mu.Unlock()
		return p, nil
	}
	c.mu.Unlock()

	body, err := c.get(ctx, "my/")
	if err != nil {
		return Profile{}, err
	}
	p, err := parseProfile(body)
	if err != nil {
		return Profile{}, err
	}
	c.mu.Lock()
	c.profile = &p
	c.mu.Unlock()
	return p, nil
}

func (c *Client) ListFolders(ctx context.Context) ([]syncpkg.RemoteFolder, error) {
	if _, err := c.Profile(ctx); err != nil {
		return nil, err
	}
	out := []syncpkg.RemoteFolder{
		{RemoteID: folderFavorites, Title: "Favourites", Kind: "favorite", Count: -1},
		{RemoteID: folderWatchLater, Title: "Watch Later", Kind: "watch_later", Count: -1},
	}
	playlists, err := c.Playlists(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range playlists {
		out = append(out, syncpkg.RemoteFolder{
			RemoteID: playlistPrefix + p.ID,
			Title:    p.Title,
			Kind:     "playlist",
			Count:    p.Count,
		})
	}
	return out, nil
}

func (c *Client) ListFolderPage(ctx context.Context, remoteID string, page, pageSize int) (syncpkg.RemotePage, error) {
	path, err := folderPath(remoteID, page)
	if err != nil {
		return syncpkg.RemotePage{}, err
	}
	body, err := c.get(ctx, path)
	if err != nil {
		return syncpkg.RemotePage{}, err
	}
	items := parseVideoCards(body)
	if len(items) == 0 && !recognizedVideoListPage(body) {
		return syncpkg.RemotePage{}, fmt.Errorf("rule34video: video list page structure not recognized")
	}
	_, pages := parsePager(body)
	hasMore := pages > page
	if pages == 0 && len(items) >= 12 {
		hasMore = true
	}
	if page > 0 && pages > 0 && page >= pages {
		hasMore = false
	}
	return syncpkg.RemotePage{Items: items, HasMore: hasMore}, nil
}

func (c *Client) Playlists(ctx context.Context) ([]Playlist, error) {
	body, err := c.get(ctx, "my/playlists/")
	if err != nil {
		return nil, err
	}
	return parsePlaylists(body)
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", c.baseURL)
	if len(c.cookies) > 0 {
		req.Header.Set("Cookie", cookieHeader(c.cookies))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rule34video: request %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("rule34video: read %s: %w", path, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusUnauthorized:
		return nil, ErrAuth
	case http.StatusForbidden, http.StatusTooManyRequests:
		return nil, ErrChallenge
	case http.StatusFound, http.StatusMovedPermanently, http.StatusSeeOther, http.StatusTemporaryRedirect:
		location := strings.ToLower(resp.Header.Get("Location"))
		if strings.Contains(location, "login") {
			return nil, ErrAuth
		}
		return nil, &HTTPError{StatusCode: resp.StatusCode, Path: path}
	default:
		return nil, &HTTPError{StatusCode: resp.StatusCode, Path: path}
	}
}

// Login submits the site's normal login form and returns the session cookies.
// The password is used only for this request.
func Login(ctx context.Context, account, password, proxyURL string) (*Credential, Profile, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return nil, Profile{}, ErrInvalidCredentials
	}
	hc, err := netproxy.NewClient(proxyURL, requestTimeout)
	if err != nil {
		return nil, Profile{}, err
	}
	defer hc.CloseIdleConnections()
	jar, err := newCookieJar()
	if err != nil {
		return nil, Profile{}, err
	}
	hc.Jar = jar
	hc.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	c := &Client{http: hc, cookies: map[string]string{}, baseURL: BaseURL + "/", userAgent: userAgent}

	body, err := c.get(ctx, "login/")
	if err != nil {
		return nil, Profile{}, err
	}
	token := csrfToken(body)
	if token == "" {
		return nil, Profile{}, ErrChallenge
	}
	form := url.Values{
		"remember_me_csrf_token": {token},
		"username":               {account},
		"pass":                   {password},
		"remember_me":            {"1"},
		"action":                 {"login"},
		"email_link":             {BaseURL + "/email/"},
		"redirect_to":            {"/"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"login/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, Profile{}, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.baseURL+"login/")
	req.Header.Set("Origin", BaseURL)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, Profile{}, fmt.Errorf("rule34video: login request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, Profile{}, ErrChallenge
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return nil, Profile{}, fmt.Errorf("rule34video: login rejected with HTTP %d", resp.StatusCode)
	}
	location := strings.ToLower(resp.Header.Get("Location"))
	if strings.Contains(location, "login") {
		return nil, Profile{}, ErrInvalidCredentials
	}

	c.cookies = cookiesFromJar(jar)
	if len(c.cookies) == 0 {
		return nil, Profile{}, ErrInvalidCredentials
	}
	profile, err := c.Profile(ctx)
	if err != nil {
		if errors.Is(err, ErrAuth) {
			return nil, Profile{}, ErrInvalidCredentials
		}
		return nil, Profile{}, err
	}
	return &Credential{Cookies: c.cookies, UserAgent: c.userAgent}, profile, nil
}

func cookiesFromJar(jar http.CookieJar) map[string]string {
	u, _ := url.Parse(BaseURL + "/")
	out := map[string]string{}
	for _, cookie := range jar.Cookies(u) {
		if strings.TrimSpace(cookie.Value) != "" {
			out[cookie.Name] = cookie.Value
		}
	}
	return out
}

func cookieHeader(cookies map[string]string) string {
	parts := make([]string, 0, len(cookies))
	for name, value := range cookies {
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, "; ")
}

func newCookieJar() (http.CookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return jar, nil
}

// DownloadOption is one official MP4 download link exposed by a video page.
type DownloadOption struct {
	Quality int
	Label   string
	URL     string
	Headers map[string]string
}

// DownloadOptions reads the public video page and returns the site's official
// download links. It performs a GET only; no login or write endpoint is used.
func (c *Client) DownloadOptions(ctx context.Context, pageURL string) ([]DownloadOption, error) {
	u, err := url.Parse(strings.TrimSpace(pageURL))
	if err != nil || u.Scheme != "https" || !sameSiteHost(u.Hostname()) || !strings.HasPrefix(u.Path, "/video/") {
		return nil, errors.New("rule34video: invalid video URL")
	}
	path := strings.TrimPrefix(u.RequestURI(), "/")
	body, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	opts := parseDownloadOptions(body)
	filtered := opts[:0]
	for _, opt := range opts {
		link, err := url.Parse(opt.URL)
		if err != nil || link.Scheme != "https" || !sameSiteHost(link.Hostname()) {
			continue
		}
		filtered = append(filtered, opt)
	}
	opts = filtered
	if len(opts) == 0 {
		return nil, errors.New("rule34video: video page has no downloadable MP4 links")
	}
	referer := u.String()
	cookie := cookieHeader(c.cookies)
	for i := range opts {
		opts[i].Headers = map[string]string{
			"Referer":    referer,
			"User-Agent": c.userAgent,
			"Accept":     "*/*",
		}
		if cookie != "" {
			opts[i].Headers["Cookie"] = cookie
		}
	}
	return opts, nil
}

func sameSiteHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "rule34video.com" || strings.HasSuffix(host, ".rule34video.com")
}
