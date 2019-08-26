package httpasset

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	// ErrNotDir is returned for a Readdir on non-directory file.
	ErrNotDir = errors.New("not a directory")

	// ErrLocateZip is returned if no trailing zip file could be detected in the binary.
	ErrLocateZip = errors.New("could not locate zip file, no end-of-central-directory signature found")
)

type opener interface {
	Open() (http.File, error)
}

type fileOpener struct {
	io.ReaderAt
	zipFile *zip.File
}

func (f fileOpener) Open() (http.File, error) {
	if f.zipFile.Method == zip.Store {
		offset, err := f.zipFile.DataOffset()
		if err != nil {
			return nil, err
		}
		return &uncompressedFile{io.NewSectionReader(f.ReaderAt, offset, int64(f.zipFile.UncompressedSize64)), f.zipFile}, nil
	}
	ff, err := f.zipFile.Open()
	if err != nil {
		return nil, err
	}
	return &compressedFile{ff, f.zipFile}, nil
}

type httpassetFS struct {
	binary io.Closer
	files  map[string]opener
}

// FileSystem implements http.FileSystem and can be closed.
type FileSystem interface {
	http.FileSystem

	// Close resources such as the embedded zip file.
	Close()
}

// Init opens the zip file appended to the binary. If the zip file could not be
// found or opened, a FileSystem pointing to the fallbackDir is returned and a log
// message is printed.
func Init(fallbackDir string) FileSystem {
	fs, err := ZipFS()
	if err == nil {
		return fs
	}
	log.Printf("%s, falling back to local directory %q", err, fallbackDir)
	return httpDir(fallbackDir)
}

type httpDir http.Dir

func (d httpDir) Open(name string) (http.File, error) {
	if !strings.HasPrefix(name, "/") {
		return nil, os.ErrNotExist
	}
	return http.Dir(d).Open(name)
}

func (d httpDir) Close() {
}

// Find end-of-directory struct, near the end of the file.
// It specifies the size & offset of the central directory.
// We assume the central directory is located just before the end-of-central-directory.
// So that allows us to calculate the original size of the zip file.
// Which in turn allows us to use godoc's zipfs to serve the zip file withend.

// Open the zip file appended to the binary and return a FileSystem that reads files from the zip file.
// If no zip file could be found, an error is returned.
func ZipFS() (FileSystem, error) {
	bin, err := binself()
	if err != nil {
		return nil, err
	}
	fi, err := bin.Stat()
	if err != nil {
		bin.Close()
		return nil, err
	}

	n := int64(65 * 1024)
	size := fi.Size()
	if size < n {
		n = size
	}
	buf := make([]byte, n)
	_, err = io.ReadAtLeast(io.NewSectionReader(bin, size-n, n), buf, len(buf))
	if err != nil {
		bin.Close()
		return nil, err
	}
	o := int64(findSignatureInBlock(buf))
	if o < 0 {
		bin.Close()
		return nil, ErrLocateZip
	}
	cdirsize := int64(binary.LittleEndian.Uint32(buf[o+12:]))
	cdiroff := int64(binary.LittleEndian.Uint32(buf[o+16:]))
	zipsize := cdiroff + cdirsize + (int64(len(buf)) - o)

	rr := io.NewSectionReader(bin, size-zipsize, zipsize)
	r, err := zip.NewReader(rr, zipsize)
	if err != nil {
		bin.Close()
		return nil, err
	}

	// Build map of files. we create our own dirs, we don't want to be dependent on zip files containing proper hierarchies.
	files := map[string]opener{}
	files[""] = dir{}
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		files[f.Name] = fileOpener{rr, f}
		elems := strings.Split(f.Name, "/")
		for e := 1; e <= len(elems)-1; e++ {
			name := strings.Join(elems[:e], "/")
			files[name] = dir{}
		}
	}
	return &httpassetFS{bin, files}, nil
}

func (fs *httpassetFS) Open(name string) (http.File, error) {
	if fs.binary == nil {
		return nil, os.ErrClosed
	}
	if !strings.HasPrefix(name, "/") {
		return nil, os.ErrNotExist
	}
	name = name[1:]
	file, ok := fs.files[name]
	if ok {
		return file.Open()
	}
	return nil, os.ErrNotExist
}

func (fs *httpassetFS) Close() {
	if fs.binary == nil {
		return
	}
	fs.binary.Close()
	fs.binary = nil
	fs.files = nil
}
