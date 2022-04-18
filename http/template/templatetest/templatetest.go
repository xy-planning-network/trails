/*

Package templatetest exposes a mock fs.FS that implements basic file operations.
Used in unit tests for the purposes of avoiding the use of testdata/ directories when unit testing template rendering.

Cribbed from Mark Bates: https://www.gopherguides.com/articles/golang-1.16-io-fs-improve-test-performance

*/
package templatetest

import (
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/xy-planning-network/trails/http/template"
)

// NewParser constructs a template.Parser with the mocked files.
func NewParser(tmpls ...FileMocker) template.Parser { return template.NewParser(NewMockFS(tmpls...)) }

type FileMocker interface {
	fs.File
	fs.FileInfo
}

type MockFS []FileMocker

func NewMockFS(tmpls ...FileMocker) fs.FS { return append(MockFS{}, tmpls...) }

// Glob checks whether the pattern matches the file after removing all directory paths from
// the respective parts.
//
// Buyer beware: Glob is a simplistic implementation of fs.GlobFS.
//
// i.e., pattern: some/long/path/*
// will match all of the following
// - some/long/path/myfile.txt
// - some/long/otherfile.txt
// - totally/different/tree/somefile.txt
// - /rootfile.txt
// ... etc.
func (mfs MockFS) Glob(pattern string) ([]string, error) {
	_, pattern = path.Split(pattern)
	matches := []string{}
	for _, f := range mfs {
		n := f.Name()
		_, filename := path.Split(n)
		matched, err := path.Match(pattern, filename)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, n)
		}
	}

	return matches, nil
}

func (mfs MockFS) Open(name string) (fs.File, error) {
	for _, f := range mfs {
		if f.Name() == name {
			return f, nil
		}
	}

	return nil, &fs.PathError{Op: "read", Path: name, Err: os.ErrNotExist}
}

type MockFile struct {
	FS      MockFS
	data    []byte
	isDir   bool
	modTime time.Time
	mode    fs.FileMode
	name    string
	size    int64
	sys     any
}

func NewMockFile(name string, data []byte) FileMocker {
	return &MockFile{data: data, name: name, size: int64(len(data))}
}

func (m *MockFile) Close() error               { return nil }
func (m *MockFile) Name() string               { return m.name }
func (m *MockFile) IsDir() bool                { return m.isDir }
func (m *MockFile) Info() (fs.FileInfo, error) { return m.Stat() }
func (m *MockFile) Mode() os.FileMode          { return m.mode }
func (m *MockFile) ModTime() time.Time         { return m.modTime }
func (m *MockFile) Size() int64                { return m.size }
func (m *MockFile) Stat() (fs.FileInfo, error) { return m, nil }
func (m *MockFile) Sys() any                   { return m.sys }
func (m *MockFile) Type() fs.FileMode          { return m.Mode().Type() }
func (m *MockFile) Read(p []byte) (int, error) {
	copy(p, m.data)
	return len(m.data), io.EOF
}
