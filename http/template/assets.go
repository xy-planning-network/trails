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
)

const (
	assetsBase = "client/dist"
)

// AssetURI encloses the environment and filesystem so when called executing a template,
// emits valid URI for client side static and bundled assets.
//
// The asset origin defaults to "/" when it is not configured and is always set with a trailing slash.
func AssetURI(origin *url.URL, env trails.Environment, filesys fs.FS) func(string) (string, error) {
	if filesys == nil {
		filesys = os.DirFS(".")
	}

	if origin == nil {
		origin, _ = url.Parse("/")
	}

	if origin.Path != "/" {
		origin.Path = "/"
	}

	return func(assetPath string) (string, error) {
		switch {
		case env.IsTesting():
			return "", nil

		default:
			// match hashed files bundled by Vite
			filename := strings.TrimSuffix(assetPath, filepath.Ext(assetPath))
			fileExt := filepath.Ext(assetPath)

			// Note: where assetPath = assets/GetDashboard.js
			// glob = client/dist/assets/GetDashboard-*.js
			glob := fmt.Sprintf("%s/%s-*%s", assetsBase, filename, fileExt)
			matches, err := fs.Glob(filesys, glob)

			if errors.Is(err, path.ErrBadPattern) {
				return "", fmt.Errorf("%w: for asset path %s", path.ErrBadPattern, assetPath)
			}

			if len(matches) > 1 {
				return "", fmt.Errorf("%w: for asset path %s", ErrMatchedAssets, assetPath)
			}

			// Note: only entry point assets like (assets/GetDashboard.js) mode will have a match
			// local development mode is not expected to match as those assets are not hashed when served by vite
			if len(matches) == 0 {
				return fmt.Sprintf("%s%s/%s", origin, assetsBase, assetPath), nil
			}

			return fmt.Sprintf("%s%s", origin, matches[0]), nil
		}
	}
}
