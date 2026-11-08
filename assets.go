//go:build !no_rule34video && (!explicit_sites || site_rule34video)

package rule34video

import (
	_ "embed"

	"magpiesync/internal/sites"
)

//go:embed assets/logo.png
var logo []byte

func init() {
	sites.RegisterAsset("rule34video", "logo.png", logo)
}
