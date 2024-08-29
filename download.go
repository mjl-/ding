package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mjl-/bstore"
)

func serveDownload(w http.ResponseWriter, r *http.Request) {
	// /dl/{release,result,file}/<reponame>/<buildid>/
	// For release & result, <name>.{zip.tgz}
	// For file, any path is allowed.
	t := strings.Split(r.URL.Path[1:], "/")
	if len(t) < 5 || hasBadElems(t) {
		http.NotFound(w, r)
		return
	}

	what := t[1]
	repoName := t[2]
	buildID, err := strconv.Atoi(t[3])
	if err != nil || repoName == "" || buildID == 0 || !(what == "release" || what == "result" || what == "file") {
		http.NotFound(w, r)
		return
	}

	fail := func(err error) {
		log.Printf("download: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}

	var repo Repo
	var b Build
	err = database.Read(r.Context(), func(tx *bstore.Tx) error {
		repo = Repo{Name: repoName}
		err := tx.Get(&repo)
		if err != nil {
			return err
		}
		b, err = bstore.QueryTx[Build](tx).FilterNonzero(Build{ID: int32(buildID), RepoName: repoName}).Get()
		return err
	})
	if err == bstore.ErrAbsent {
		http.NotFound(w, r)
		return
	} else if err != nil {
		fail(err)
		return
	}

	if what == "file" {
		filename := fmt.Sprintf("%s/build/%s/%d/dl/%s", dingDataDir, repoName, buildID, path.Join(t[4:]...))
		http.ServeFile(w, r, filename)
		return
	}

	if len(t) != 5 {
		http.NotFound(w, r)
		return
	}

	files := []archiveFile{}
	if b.Released == nil && what == "release" {
		http.NotFound(w, r)
		return
	}
	for _, res := range b.Results {
		var p string
		if what == "release" {
			p = fmt.Sprintf("%s/release/%s/%d/%s", dingDataDir, repoName, buildID, filepath.Base(res.Filename))
		} else {
			p = fmt.Sprintf("%s/build/%s/%d/checkout/%s/%s", dingDataDir, repoName, buildID, repo.CheckoutPath, res.Filename)
		}
		files = append(files, archiveFile{p, res.Filesize})
	}
	if len(files) == 0 {
		http.NotFound(w, r)
		return
	}
	name := t[4]
	isGzip := what == "release"
	serveDownload0(w, r, name, files, isGzip)
}

type archiveFile struct {
	Path string
	Size int64
}

// We have .gz on disk.  gzip is a deflate stream with a header and a footer.
// A zip file consists of headers/footer and deflate streams.
// So we can serve zip files quickly based on the .gz's on disk.
// We just have to strip the gzip header & footer and pass the raw deflate stream through.
type gzipStrippingDeflateWriter struct {
	w        io.Writer
	header   []byte // We need the 10 byte header before we can do anything.
	flag     byte   // From header, indicates optional fields we must skip. We clear the flags one we skipped parts.
	leftover []byte // We always hold the last 8 bytes back, it could be the gzip footer that we must skip.
}

func (x *gzipStrippingDeflateWriter) Write(buf []byte) (int, error) {
	n := len(buf)

	if len(x.header) < 10 {
		take := 10 - len(x.header)
		if take > len(buf) {
			take = len(buf)
		}
		x.header = append(x.header, buf[:take]...)
		buf = buf[take:]
		if len(x.header) == 10 {
			if x.header[0] != 0x1f || x.header[1] != 0x8b {
				return -1, fmt.Errorf("not a gzip header: %x", x.header[:2])
			}
			x.flag = x.header[3]
		}
	}
	const (
		FlagFHCRC = 1 << iota
		FlagFEXTRA
		FlagFNAME
		FlagFCOMMENT
	)

	// Null-terminated string.
	skipString := func(l []byte) ([]byte, bool) {
		for i := range l {
			if l[i] == 0 {
				return l[i+1:], true
			}
		}
		return nil, false
	}

	if (x.flag & FlagFEXTRA) != 0 {
		return -1, fmt.Errorf("extra gzip header data, not supported yet") // please fix me (:
	}
	if (x.flag & FlagFNAME) != 0 {
		var skipped bool
		buf, skipped = skipString(buf)
		if skipped {
			x.flag &^= FlagFNAME
		}
	}
	if (x.flag & FlagFCOMMENT) != 0 {
		var skipped bool
		buf, skipped = skipString(buf)
		if skipped {
			x.flag &^= FlagFCOMMENT
		}
	}

	if (x.flag & FlagFHCRC) != 0 {
		// 2 bytes

		if len(x.leftover)+len(buf) < 2 {
			x.leftover = append(x.leftover, buf...)
			return n, nil
		}
		drop := 2
		if len(x.leftover) > 0 {
			xdrop := drop
			if len(x.leftover) < drop {
				xdrop = len(x.leftover)
			}
			drop -= xdrop
			x.leftover = x.leftover[xdrop:]
		}
		buf = buf[:drop]
		x.flag &^= FlagFHCRC
	}

	if len(buf) < 8 {
		nn := 8 - len(buf)
		if nn > len(x.leftover) {
			nn = len(x.leftover)
		}
		if nn > 0 {
			_, err := x.w.Write(x.leftover[:nn])
			if err != nil {
				return -1, err
			}
			x.leftover = x.leftover[nn:]
		}
		x.leftover = append(x.leftover, buf...)
		return n, nil
	}
	// Below here, we have at least 8 bytes in buf.

	if len(x.leftover) > 0 {
		_, err := x.w.Write(x.leftover)
		if err != nil {
			return -1, err
		}
		x.leftover = nil
	}
	if len(buf) > 8 {
		_, err := x.w.Write(buf[:len(buf)-8])
		if err != nil {
			return -1, err
		}
	}
	x.leftover = append(x.leftover, buf[len(buf)-8:]...)
	return n, nil
}

func (x *gzipStrippingDeflateWriter) Close() error {
	if len(x.leftover) != 8 {
		return fmt.Errorf("not 8 bytes left over at close")
	}
	return nil
}

func newGzipStrippingDeflateWriter(w io.Writer) (io.WriteCloser, error) {
	return &gzipStrippingDeflateWriter{w: w}, nil
}

// `files` do not have the .gz suffix they have in the file system.
func serveDownload0(w http.ResponseWriter, r *http.Request, name string, files []archiveFile, isGzip bool) {
	if strings.HasSuffix(name, ".zip") {
		base := strings.TrimSuffix(name, ".zip")
		w.Header().Set("Content-Type", "application/zip")
		zw := zip.NewWriter(w)
		if isGzip {
			zw.RegisterCompressor(zip.Deflate, newGzipStrippingDeflateWriter)
		}

		addFile := func(file archiveFile) bool {
			lpath := file.Path
			if isGzip {
				lpath += ".gz"
			}
			f, err := os.Open(lpath)
			if err != nil {
				log.Printf("download: open %s to add to zip: %s", lpath, err)
				return false
			}
			defer f.Close()

			filename := path.Base(file.Path)
			fw, err := zw.Create(base + "/" + filename)
			if err != nil {
				log.Printf("download: adding file to zip: %s", err)
				return false
			}
			_, err = io.Copy(fw, f)
			if err != nil {
				// Probably just a closed connection.
				log.Printf("download: copying data: %s", err)
				return false
			}
			return true
		}
		for _, path := range files {
			if !addFile(path) {
				break
			}
		}
		// Errors would probably be closed connections.
		err := zw.Close()
		if err != nil {
			log.Printf("download: finishing write: %s", err)
		}
	} else if strings.HasSuffix(name, ".tgz") {
		base := strings.TrimSuffix(name, ".tgz")
		gzw := gzip.NewWriter(w)
		tw := tar.NewWriter(gzw)

		addFile := func(file archiveFile) bool {
			lpath := file.Path
			if isGzip {
				lpath += ".gz"
			}
			f, err := os.Open(lpath)
			if err != nil {
				log.Printf("download: open %s to add to tgz: %s", lpath, err)
				return false
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				log.Printf("download: stat %s to add to tgz: %s", lpath, err)
				return false
			}
			var gzr io.Reader = f
			if isGzip {
				gzr, err = gzip.NewReader(f)
				if err != nil {
					log.Printf("download: reading gzip %s: %s", lpath, err)
					return false
				}
			}

			hdr := &tar.Header{
				Name:     base + "/" + path.Base(file.Path),
				Mode:     int64(fi.Mode().Perm()),
				Size:     file.Size,
				ModTime:  fi.ModTime(),
				Typeflag: tar.TypeReg,
			}
			err = tw.WriteHeader(hdr)
			if err != nil {
				log.Printf("download: adding file to tgz: %s", err)
				return false
			}
			_, err = io.Copy(tw, gzr)
			return err == nil
		}
		for _, path := range files {
			if !addFile(path) {
				break
			}
		}
		// Errors would probably be closed connections.
		tw.Close()
		gzw.Close()
	} else {
		http.NotFound(w, r)
		return
	}
}
