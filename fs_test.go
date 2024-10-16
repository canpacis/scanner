package scanner_test

import (
	"bytes"
	"io/fs"
	"time"
)

type FileInfo struct {
	name    string
	buf     *bytes.Buffer
	mode    fs.FileMode
	modTime time.Time
}

func (fi *FileInfo) Name() string {
	return fi.name
}

func (fi *FileInfo) Size() int64 {
	return int64(len(fi.buf.Bytes()))
}

func (fi *FileInfo) Mode() fs.FileMode {
	return fi.mode
}

func (fi *FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *FileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (fi *FileInfo) Sys() any {
	return nil
}

type File struct {
	info   *FileInfo
	closed bool
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *File) Read(p []byte) (int, error) {
	return f.info.buf.Read(p)
}

func (f *File) Close() error {
	if f.closed {
		return fs.ErrClosed
	}

	f.closed = true
	return nil
}

func NewFile(name string, content []byte) *File {
	return &File{
		info: &FileInfo{
			name:    name,
			mode:    fs.ModePerm,
			buf:     bytes.NewBuffer(content),
			modTime: time.Now(),
		},
	}
}

type FS struct {
	Files map[string]*File
}

func (fsys FS) Open(name string) (fs.File, error) {
	file, ok := fsys.Files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return file, nil
}

func (fsys FS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries := []fs.DirEntry{}

	for _, file := range fsys.Files {
		entries = append(entries, &DirEntry{file: file})
	}

	return entries, nil
}

type DirEntry struct {
	file *File
}

func (e *DirEntry) Name() string {
	return e.file.info.name
}

func (e *DirEntry) IsDir() bool {
	return e.file.info.IsDir()
}

func (e *DirEntry) Type() fs.FileMode {
	return e.file.info.mode
}

func (e *DirEntry) Info() (fs.FileInfo, error) {
	return e.file.info, nil
}
