package template

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/xy-planning-network/trails"
)

const (
	assetsBase = "client/dist"
)

// AssetURI encloses the environment and filesystem so when called executing a template,
// emits valid URI for client side static and bundled assets.
//
// TODO:
//
// - configurable OriginHost would allow for support of a CDN and custom Vite server origin
func AssetURI(env trails.Environment, filesys fs.FS) func(string) string {
	if filesys == nil {
		filesys = os.DirFS(".")
	}

	return func(assetPath string) string {
		switch {
		case env.IsTesting():
			return ""

		case env.IsDevelopment():
			return fmt.Sprintf("http://localhost:8080/%s/%s", assetsBase, assetPath)

		default:
			// match hashed files bundled by Vite
			filename := strings.TrimSuffix(assetPath, filepath.Ext(assetPath))
			fileExt := filepath.Ext(assetPath)

			// Note: where assetPath = assets/GetDashboard.js
			// glob = client/dist/assets/GetDashboard-*.js
			glob := fmt.Sprintf("%s/%s-*%s", assetsBase, filename, fileExt)
			matches, err := fs.Glob(filesys, glob)

			if errors.Is(err, path.ErrBadPattern) || len(matches) == 0 {
				return fmt.Sprintf("/%s/%s", assetsBase, assetPath)
			}

			return fmt.Sprintf("/%s", matches[0])
		}
	}
}
