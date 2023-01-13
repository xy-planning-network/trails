package template

import (
	"fmt"
	"io/fs"
)

// A mergeFS maps filepaths to fs.Open functions,
// caching which html template to open during runtime.
// A mergeFS is read-only at runtime,
// disallowing adding or replacing fs.Open functions at runtime.
//
// mergeFS will not fallback to files in another fs.FS
// if a cached entry becomes invalid.
// e.g., removing a file in an OS-level filesystem will not fallback
// to another filesystem during runtime.
//
// mergeFS implements fs.FS
type mergeFS map[string]func(string) (fs.File, error)

// Open opens the file by name from the merged FS.
func (mfs mergeFS) Open(name string) (fs.File, error) {
	fn, ok := mfs[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, name)
	}

	return fn(name)
}

// merge combines the fses into a single cache,
// constructing a mergeFS.
// merge adds entries in reverse order,
// meaning if more than one fs.FS references the same filepath,
// the first reference is cached.
func merge(fses []fs.FS) mergeFS {
	mfs := make(map[string]func(string) (fs.File, error))
	for i := len(fses) - 1; i >= 0; i-- {
		fs.WalkDir(fses[i], ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || (d != nil && d.IsDir()) {
				return nil
			}

			mfs[path] = fses[i].Open
			return nil
		})
	}

	return mfs
}
