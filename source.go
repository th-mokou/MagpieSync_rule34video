//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"context"

	"magpiesync/internal/sites"
	rule34 "magpiesync/internal/sites/plugins/rule34video/core"
	syncpkg "magpiesync/internal/sync"
)

func (driver) NewSource(_ context.Context, in sites.AccountContext) (syncpkg.Source, error) {
	cred, err := rule34.ParseCredential(in.Credential)
	if err != nil {
		return nil, err
	}
	client, err := rule34.NewClientWithUserAgent(cred.Cookies, in.ProxyURL, cred.UserAgent)
	if err != nil {
		return nil, err
	}
	return rule34.NewAdapter(client), nil
}

func (driver) NewDirectResolver(_ context.Context, in sites.AccountContext) (sites.DirectResolver, error) {
	cred, err := rule34.ParseCredential(in.Credential)
	if err != nil {
		return nil, err
	}
	client, err := rule34.NewClientWithUserAgent(cred.Cookies, in.ProxyURL, cred.UserAgent)
	if err != nil {
		return nil, err
	}
	return &directResolver{client: client}, nil
}

type directResolver struct {
	client *rule34.Client
}

func (r *directResolver) Options(ctx context.Context, pageURL string) ([]sites.DirectOption, error) {
	opts, err := r.client.DownloadOptions(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	out := make([]sites.DirectOption, 0, len(opts))
	for _, o := range opts {
		out = append(out, sites.DirectOption{Quality: o.Quality, Label: o.Label, URL: o.URL, Headers: o.Headers})
	}
	return out, nil
}

func (r *directResolver) Close() error { return nil }
