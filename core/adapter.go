//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	"context"

	syncpkg "magpiesync/internal/sync"
)

type Adapter struct {
	client *Client
}

func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

func (a *Adapter) ListFolders(ctx context.Context) ([]syncpkg.RemoteFolder, error) {
	return a.client.ListFolders(ctx)
}

func (a *Adapter) ListFolderPage(ctx context.Context, remoteID string, page, pageSize int) (syncpkg.RemotePage, error) {
	return a.client.ListFolderPage(ctx, remoteID, page, pageSize)
}

func (a *Adapter) Close() error { return a.client.Close() }
