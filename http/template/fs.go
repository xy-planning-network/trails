package template

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sync"
)

// mergeFS implements fs.FS
type mergeFS struct {
	// A cache for minimizing ascertaining which directory holds the template.
	cache map[string]func(string) (fs.File, error)

	// Current working directory, or, embedded filesystem
	userDir fs.FS

	// Package-level directory embedding tmpl/
	pkgDir fs.FS

	sync.Mutex
}

// Open opens the file matching the name using the following strategy:
// - check the cache
// - check the OS filesystem
// - check the package-level virtual filesystem
//
// Whenever a file is found and is not present in the cache, it is added.
// Nothing removes references from the cache.
//
// The cache cannot become invalid at runtime since pkgDir is embedded.
// If a file is removed from the OS during runtime,
// then a reference to it from the cache returns the same error (fs.ErrNotExist)
// as if the cache did not have that reference.
func (mfs *mergeFS) Open(name string) (fs.File, error) {
	// NOTE(dlk): while a concurrent routine could add a reference
	// to the cache before this returns,
	// let's err on the side of performance and not have this function
	// blocking while waiting to read and only block when needing to write.
	fn, ok := mfs.cache[name]
	if ok {
		return fn(name)
	}

	file, err := mfs.userDir.Open(name)
	if err == nil {
		mfs.Lock()
		mfs.cache[name] = mfs.userDir.Open
		mfs.Unlock()

		return file, nil
	}

	var pe *fs.PathError
	if errors.As(err, &pe) && (errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid)) {
		file, err = mfs.pkgDir.Open(name)
		if err != nil {
			return nil, fmt.Errorf("could not open template %s: %s", name, err)
		}

		mfs.Lock()
		mfs.cache[name] = mfs.pkgDir.Open
		mfs.Unlock()
		return file, nil
	}

	return nil, fmt.Errorf("unable to open template: %w", err)
}

//go:embed tmpl/*
var pkgFS embed.FS
