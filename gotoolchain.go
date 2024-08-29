package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
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
	case "go", "go-prev":
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
	case "go", "go-prev":
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
