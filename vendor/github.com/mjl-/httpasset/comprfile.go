package httpasset

import (
	"archive/zip"
	"errors"
	"io"
	"net/http"
	"os"
)

var (
	errCompressedSeek = errors.New("seek on compressed file")
)

type compressedFile struct {
	io.ReadCloser
	zipFile *zip.File
}

var _ http.File = &compressedFile{}

func (f *compressedFile) Seek(offset int64, whence int) (int64, error) {
	return -1, errCompressedSeek
}

func (f *compressedFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, ErrNotDir
}

func (f *compressedFile) Stat() (os.FileInfo, error) {
	return f.zipFile.FileInfo(), nil
}
