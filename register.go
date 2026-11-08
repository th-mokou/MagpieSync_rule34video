//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"magpiesync/internal/sites"
	rule34 "magpiesync/internal/sites/plugins/rule34video/core"
)

func init() {
	sites.Register(sites.Site{
		ID:       "rule34video",
		Name:     "Rule 34 Video",
		Homepage: "https://rule34video.com",
		Logo:     "/api/site-assets/rule34video/logo.png",
		Ready:    true,
		Download: true,
		DownloadQualities: []sites.DownloadQuality{
			{Value: 1080, Label: "1080P"},
			{Value: 720, Label: "720P"},
			{Value: 480, Label: "480P"},
			{Value: 360, Label: "360P"},
		},
		Categories: []sites.Category{
			{ID: "favorite", Name: "Favourites"},
			{ID: "watch_later", Name: "Watch Later"},
			{ID: "playlist", Name: "Playlists"},
		},
		Naming: &sites.SiteNaming{
			VideoTemplate: "{{title}}",
			PageTemplate:  "{{page_title}}",
			TemplateVars:  []string{"id", "title", "page_title", "page_no"},
		},
		Login: sites.LoginConfig{
			Hint: "使用 Rule 34 Video 的邮箱或用户名和密码登录。建议同时配置代理。",
			Methods: []sites.LoginMethod{
				{ID: "password", Label: "账号密码"},
				{ID: "cookie", Label: "使用 Cookie"},
			},
			Password: &sites.PasswordConfig{
				UsernameLabel: "账号或邮箱",
				PasswordLabel: "密码",
			},
			Cookie: &sites.CookieConfig{
				Placeholder: "PHPSESSID=...; __Host-kt_member_v2=...",
				Help:        "从已登录的浏览器复制完整 Cookie；至少需要站点主会话字段，遇到 DDoS-Guard/Cloudflare 验证时还需要同一出口 IP 下的验证 Cookie。",
				RequiresUA:  true,
				UAHelp:      "必须与生成 Cookie 的浏览器一致；使用无浏览器模式时，Cookie 过期后需要重新复制。",
			},
		},
		DownloadInfo: sites.DownloadConfig{
			ActorRole:    "作者",
			ImageReferer: "https://rule34video.com/",
		},
	})
	sites.RegisterDriver("rule34video", driver{})
}

type driver struct{}

func (driver) SiteID() string { return "rule34video" }

func (driver) IsAuth(err error) bool { return rule34.IsAuth(err) }

func (driver) IsFatal(err error) bool { return rule34.IsFatal(err) }
