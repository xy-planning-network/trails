package template

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/xy-planning-network/trails"
	"github.com/xy-planning-network/trails/logger"
)

const (
	assetsBase = "client/dist"
)

// AssetURI encloses the environment and filesystem so when called executing a template,
// emits valid URI for client side static and bundled assets.
//
// The asset origin defaults to "/" when it is not configured and is always set with a trailing slash.
func AssetURI(origin *url.URL, env trails.Environment, filesys fs.FS, l logger.Logger) func(string) string {
	if filesys == nil {
		filesys = os.DirFS(".")
	}

	if origin == nil {
		origin, _ = url.Parse("/")
	}

	if origin.Path != "/" {
		origin.Path = "/"
	}

	return func(assetPath string) string {
		switch {
		case env.IsTesting():
			return ""

		default:
			// match hashed files bundled by Vite
			filename := strings.TrimSuffix(assetPath, filepath.Ext(assetPath))
			fileExt := filepath.Ext(assetPath)

			// Note: where assetPath = assets/GetDashboard.js
			// glob = client/dist/assets/GetDashboard-*.js
			glob := fmt.Sprintf("%s/%s-*%s", assetsBase, filename, fileExt)
			matches, err := fs.Glob(filesys, glob)

			// Note: when in local dev mode it is expected that patterns won't match
			if errors.Is(err, path.ErrBadPattern) || len(matches) == 0 {
				return fmt.Sprintf("%s%s/%s", origin, assetsBase, assetPath)
			}

			if len(matches) > 1 {
				l.Error("Asset path found multiple matches.", &logger.LogContext{
					Data: map[string]any{
						"assetPath": assetPath,
					},
				})
			}

			return fmt.Sprintf("%s%s", origin, matches[0])
		}
	}
}
