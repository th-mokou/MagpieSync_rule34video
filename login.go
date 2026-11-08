//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"magpiesync/internal/cookieutil"
	"magpiesync/internal/sites"
	rule34 "magpiesync/internal/sites/plugins/rule34video/core"
)

func (driver) CookieLogin(ctx context.Context, in sites.CookieInput) (*sites.LoginResult, error) {
	cookies := cookieutil.Parse(in.RawCookie)
	for name, value := range cookies {
		value = strings.TrimSpace(value)
		if value == "" {
			delete(cookies, name)
			continue
		}
		cookies[name] = value
	}
	if cookies["PHPSESSID"] == "" && cookies["__Host-kt_member_v2"] == "" {
		return nil, &sites.AuthError{Kind: sites.AuthBadRequest, Message: "PHPSESSID or __Host-kt_member_v2 is required"}
	}
	client, err := rule34.NewClientWithUserAgent(cookies, in.ProxyURL, in.UserAgent)
	if err != nil {
		return nil, &sites.AuthError{Kind: sites.AuthBadRequest, Message: err.Error(), Err: err}
	}
	defer client.Close()
	profile, err := client.Profile(ctx)
	if err != nil {
		slog.Warn("rule34video profile check failed", "site", "rule34video", "cookies", cookieNames(cookies), "error", err)
		if rule34.IsAuth(err) {
			return nil, &sites.AuthError{Kind: sites.AuthUnauthorized, Message: "Cookie 无效或已过期", Err: err}
		}
		return nil, &sites.AuthError{Kind: sites.AuthBadGateway, Message: err.Error(), Err: err}
	}
	uid := strings.TrimSpace(profile.UID)
	name := strings.TrimSpace(profile.Name)
	if uid == "" || !isDigits(uid) {
		return nil, errors.New("Rule 34 Video 未返回数字账户 ID")
	}
	if name == "" {
		name = uid
	}
	raw, err := json.Marshal(rule34.Credential{Cookies: cookies, UserAgent: in.UserAgent})
	if err != nil {
		return nil, err
	}
	return &sites.LoginResult{
		UID:        uid,
		Name:       name,
		AvatarURL:  profile.AvatarURL,
		Credential: raw,
		AutoSync:   true,
	}, nil
}

func (driver) PasswordLogin(ctx context.Context, in sites.PasswordInput) (*sites.LoginResult, error) {
	cred, profile, err := rule34.Login(ctx, in.Email, in.Password, in.ProxyURL)
	if err != nil {
		slog.Warn("rule34video password login failed", "error", err)
		switch {
		case errors.Is(err, rule34.ErrInvalidCredentials):
			return nil, &sites.AuthError{Kind: sites.AuthUnauthorized, Message: "邮箱/用户名或密码错误", Err: err}
		case errors.Is(err, rule34.ErrChallenge):
			return nil, &sites.AuthError{Kind: sites.AuthBadGateway, Message: "站点验证或限流未通过；请确认代理可用后重试", Err: err}
		case errors.Is(err, context.DeadlineExceeded):
			return nil, &sites.AuthError{Kind: sites.AuthTimeout, Message: "登录超时，请重试", Err: err}
		default:
			return nil, &sites.AuthError{Kind: sites.AuthBadGateway, Message: err.Error(), Err: err}
		}
	}
	raw, err := json.Marshal(cred)
	if err != nil {
		return nil, err
	}
	uid := strings.TrimSpace(profile.UID)
	name := strings.TrimSpace(profile.Name)
	if uid == "" || !isDigits(uid) {
		return nil, errors.New("Rule 34 Video 未返回数字账户 ID")
	}
	if name == "" {
		name = uid
	}
	return &sites.LoginResult{
		UID:        uid,
		Name:       name,
		AvatarURL:  profile.AvatarURL,
		Credential: raw,
		AutoSync:   true,
	}, nil
}

func isDigits(s string) bool {
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

func cookieNames(cookies map[string]string) []string {
	names := make([]string, 0, len(cookies))
	for name := range cookies {
		names = append(names, name)
	}
	return names
}
