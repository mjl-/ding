package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/mjl-/goreleases"
)

func installGoToolchain(file goreleases.File, shortname string) error {
	slog.Debug("installing go toolchain", "filename", file.Filename)

	if config.GoToolchainDir == "" {
		return fmt.Errorf("go toolchain dir not configured")
	}
	if !validGoversion(file.Version) {
		return fmt.Errorf("bad goversion")
	}

	perms, err := makeGoreleasesPermissions()
	if err != nil {
		return err
	}
	tmpdir, err := os.MkdirTemp(config.GoToolchainDir, "tmp-"+file.Version)
	if err != nil {
		return fmt.Errorf("creating temp dir for downloading")
	}
	defer func() {
		// Dir will be empty on success: the go directory will have been moved to the go toolchain dir.
		os.RemoveAll(tmpdir)
	}()
	err = goreleases.Fetch(file, tmpdir, perms)
	if err != nil {
		return fmt.Errorf("fetching toolchain from golang.org/dl/: %v", err)
	}

	versionDst := path.Join(config.GoToolchainDir, file.Version)
	err = os.Rename(path.Join(tmpdir, "go"), versionDst)
	if err != nil {
		return fmt.Errorf("moving from tmp dir to final destination: %v", err)
	}

	switch shortname {
	case "go", "go-prev", "go-next":
		err := makeToolchainSymlink(file.Version, shortname, perms)
		if err != nil {
			return fmt.Errorf("activating toolchain under shortname: %v", err)
		}
	case "":
	default:
		return fmt.Errorf("invalid shortname")
	}
	return nil
}

func validGoversion(goversion string) bool {
	if !strings.HasPrefix(goversion, "go") || strings.Contains(goversion, "/") || strings.Contains(goversion, "..") || strings.Contains(goversion, "\\") {
		return false
	}
	return true
}

func makeGoreleasesPermissions() (*goreleases.Permissions, error) {
	if !config.IsolateBuilds.Enabled {
		return nil, nil
	}

	fi, err := os.Stat(config.GoToolchainDir)
	if err != nil {
		return nil, fmt.Errorf("state on go toolchain dir: %v", err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("cannot find uid/gid of go toolchain dir due to stat")
	}
	perms := &goreleases.Permissions{
		Uid:  int(st.Uid),
		Gid:  int(st.Gid),
		Mode: fi.Mode() &^ os.ModeType,
	}
	return perms, nil
}

func removeGoToolchain(goversion string) error {
	slog.Debug("removing go toolchain", "goversion", goversion)

	// Verify sanity of request.
	if config.GoToolchainDir == "" {
		return fmt.Errorf("go toolchain dir not configured")
	}
	if !validGoversion(goversion) {
		return fmt.Errorf("bad goversion")
	}

	// Return helpful error diagnostic when toolchain isn't there.
	_, err := os.Stat(path.Join(config.GoToolchainDir, goversion))
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("version is not installed")
	}

	// Remove well-known symlinks that point to the toolchain that will be removed.
	var active GoToolchains
	active.Go, _ = os.Readlink(path.Join(config.GoToolchainDir, "go"))
	active.GoPrev, _ = os.Readlink(path.Join(config.GoToolchainDir, "go-prev"))
	active.GoNext, _ = os.Readlink(path.Join(config.GoToolchainDir, "go-next"))
	removeLink := func(short, v string) {
		if v == goversion {
			if err := os.Remove(path.Join(config.GoToolchainDir, short)); err != nil {
				slog.Error("remove go version symlink from toolchaindir", "err", err, "shortname", short)
			}
		}
	}
	removeLink("go", active.Go)
	removeLink("go-prev", active.GoPrev)
	removeLink("go-next", active.GoNext)

	return os.RemoveAll(path.Join(config.GoToolchainDir, goversion))
}

func activateGoToolchain(goversion, shortname string) error {
	slog.Debug("activating go toolchain", "goversion", goversion, "shortname", shortname)

	// Verify sanity of request.
	if config.GoToolchainDir == "" {
		return fmt.Errorf("go toolchain dir not configured")
	}
	if !validGoversion(goversion) {
		return fmt.Errorf("bad goversion")
	}
	switch shortname {
	case "go", "go-prev", "go-next":
	default:
		return fmt.Errorf("bad shortname")
	}

	// Ensure requested toolchain is present.
	_, err := os.Stat(path.Join(config.GoToolchainDir, goversion))
	if err != nil {
		return fmt.Errorf("stat on requested toolchain: %v", err)
	}

	// Make the symlink.
	perms, err := makeGoreleasesPermissions()
	if err != nil {
		return err
	}
	return makeToolchainSymlink(goversion, shortname, perms)
}

func makeToolchainSymlink(goversion, shortname string, perms *goreleases.Permissions) error {
	// Ignore error for removal, there may not be any active toolchain, that's fine.
	shortpath := path.Join(config.GoToolchainDir, shortname)
	os.Remove(shortpath)

	// Make the symlink, and set permissions.
	err := os.Symlink(goversion, shortpath)
	if err != nil {
		return fmt.Errorf("creating symlink for toolchain to new version: %v", err)
	}
	if perms != nil {
		err = os.Lchown(shortpath, perms.Uid, perms.Gid)
		if err != nil {
			return fmt.Errorf("chown on symlink for toolchain to new version: %v", err)
		}
	}
	return nil
}

func automaticGoToolchain() (updated bool, rerr error) {
	// Current Go toolchains.
	var active GoToolchains
	active.Go, _ = os.Readlink(path.Join(config.GoToolchainDir, "go"))
	active.GoPrev, _ = os.Readlink(path.Join(config.GoToolchainDir, "go-prev"))
	active.GoNext, _ = os.Readlink(path.Join(config.GoToolchainDir, "go-next"))

	// todo: ideally get signal in a cheaper way. the full releases json is over one megabyte of json data.
	releases, err := goreleases.ListAll()
	if err != nil {
		return false, fmt.Errorf("listing go toolchains: %v", err)
	}

	type version struct {
		minor  int
		patch  int
		rc     int
		stable bool
	}
	type release struct {
		rel     goreleases.Release
		version version
	}
	var rels []release

	// go1.X(.Y|rcZ)
	for _, r := range releases {
		if !strings.HasPrefix(r.Version, "go1.") || r.Version == "go1.9.2rc2" {
			continue
		}
		v := r.Version[len("go1."):]
		// Read minor.
		n := 0
		for _, c := range v {
			if c >= '0' && c <= '9' {
				n++
			} else {
				break
			}
		}
		if n == 0 {
			continue
		}
		minor, err := strconv.ParseInt(v[:n], 10, 32)
		if err != nil {
			slog.Error("parsing go version number, bad minor", "version", r.Version, "err", err)
			continue
		}
		v = v[n:]
		var patch, rc int64
		if v == "" {
			patch = 0
		} else if strings.HasPrefix(v, "rc") || strings.HasPrefix(v, "beta") {
			if strings.HasPrefix(v, "rc") {
				v = v[2:]
			} else {
				v = v[4:]
			}
			rc, err = strconv.ParseInt(v, 10, 32)
			if err != nil {
				slog.Error("parsing go version number, bad release candidate", "version", r.Version, "err", err)
				continue
			}
		} else if !strings.HasPrefix(v, ".") {
			slog.Error("parsing go version number, no dot after minor", "version", r.Version)
			continue
		} else if patch, err = strconv.ParseInt(v[1:], 10, 32); err != nil {
			slog.Error("parsing go version number, bad patch", "version", r.Version, "err", err)
			continue
		}

		if r.Stable != (rc == 0) {
			slog.Error("release version number and stable field mismatch", "version", r.Version, "rc", rc, "stable", r.Stable)
			continue
		}

		rels = append(rels, release{r, version{int(minor), int(patch), int(rc), r.Stable}})
	}

	// Sort newest first.
	sort.Slice(rels, func(i, j int) bool {
		a, b := rels[i].version, rels[j].version
		if a.minor != b.minor {
			return a.minor > b.minor
		}
		if a.rc != b.rc {
			return a.rc > b.rc
		}
		return a.patch > b.patch
	})

	// First stable is "go", first unstable is "gonext" but only if newer than "go".
	// "goprev" is first stable that isn't of same minor as "go".
	var gocur, gonext, goprev *release
	for i, rel := range rels {
		if rel.version.stable {
			if gocur == nil {
				gocur = &rels[i]
			} else if rel.version.minor < gocur.version.minor {
				goprev = &rels[i]
				break
			}
		} else if gonext == nil && !rel.version.stable && (gocur == nil || gocur.version.minor < rel.version.minor) {
			gonext = &rels[i]
		}
	}
	if gocur == nil || goprev == nil {
		return false, fmt.Errorf("did not find current or previous go release")
	}
	if gonext != nil && gonext.version.minor == gocur.version.minor {
		gonext = nil
	}

	// Check if we are already at the desired state.
	nactive := GoToolchains{
		Go:     gocur.rel.Version,
		GoPrev: goprev.rel.Version,
	}
	if gonext != nil {
		nactive.GoNext = gonext.rel.Version
	}
	if nactive == active {
		slog.Debug("go toolchains already at desired versions", "active", active)
		return false, nil
	}

	slog.Debug("updating go toolchains", "active", active, "nactive", nactive)

	// List installed toolchains, to see which we are missing.
	var installed []string
	files, err := os.ReadDir(config.GoToolchainDir)
	if err != nil {
		return false, fmt.Errorf("listing files in go toolchain dir")
	}
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "go") {
			installed = append(installed, f.Name())
		}
	}

	ensureInstalled := func(r *release) error {
		if slices.Contains(installed, r.rel.Version) {
			return nil
		}
		file, err := goreleases.FindFile(r.rel, runtime.GOOS, runtime.GOARCH, "archive")
		if err != nil {
			return err
		}
		return installGoToolchain(file, "")
	}

	// Ensure needed toolchains are installed.
	if err := ensureInstalled(gocur); err != nil {
		return false, err
	}
	if err := ensureInstalled(goprev); err != nil {
		return false, err
	}
	if gonext != nil {
		if err := ensureInstalled(gonext); err != nil {
			return false, err
		}
	}

	// Set new symlinks.
	perms, err := makeGoreleasesPermissions()
	if err != nil {
		return false, err
	}
	if err := makeToolchainSymlink(gocur.rel.Version, "go", perms); err != nil {
		return false, err
	}
	if err := makeToolchainSymlink(goprev.rel.Version, "go-prev", perms); err != nil {
		return false, err
	}
	if gonext != nil {
		if err := makeToolchainSymlink(gonext.rel.Version, "go-next", perms); err != nil {
			return false, err
		}
	} else {
		if err := os.Remove(path.Join(config.GoToolchainDir, "go-next")); err != nil {
			slog.Error("removing go-next symlink", "err", err)
		}
	}

	return true, nil
}
