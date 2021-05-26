/*

Cribbed from Mark Bates: https://www.gopherguides.com/articles/golang-1.16-io-fs-improve-test-performance

*/
package templatetest

import (
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/xy-planning-network/trails/http/template"
)

func NewParser(tmpls ...*MockTmpl) template.Parser { return template.NewParser(NewMockFS(tmpls...)) }

type MockFS []*MockTmpl

func NewMockFS(tmpls ...*MockTmpl) fs.FS { return append(MockFS{}, tmpls...) }

func (mfs MockFS) Open(name string) (fs.File, error) {
	for _, f := range mfs {
		if f.Name() == name {
			return f, nil
		}
	}

	return nil, &fs.PathError{Op: "read", Path: name, Err: os.ErrNotExist}
}

type MockTmpl struct {
	FS      MockFS
	data    []byte
	isDir   bool
	modTime time.Time
	mode    fs.FileMode
	name    string
	size    int64
	sys     interface{}
}

func NewMockTmpl(name string, data []byte) *MockTmpl {
	return &MockTmpl{data: data, name: name, size: int64(len(data))}
}

func (m *MockTmpl) Close() error               { return nil }
func (m *MockTmpl) Name() string               { return m.name }
func (m *MockTmpl) IsDir() bool                { return m.isDir }
func (m *MockTmpl) Info() (fs.FileInfo, error) { return m.Stat() }
func (m *MockTmpl) Mode() os.FileMode          { return m.mode }
func (m *MockTmpl) ModTime() time.Time         { return m.modTime }
func (m *MockTmpl) Size() int64                { return m.size }
func (m *MockTmpl) Stat() (fs.FileInfo, error) { return m, nil }
func (m *MockTmpl) Sys() interface{}           { return m.sys }
func (m *MockTmpl) Type() fs.FileMode          { return m.Mode().Type() }
func (m *MockTmpl) Read(p []byte) (int, error) {
	copy(p, m.data)
	return len(m.data), io.EOF
}
