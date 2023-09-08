package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type NSOverlay struct {
	Path string
}

func MergeOverlay(dir, tfPath, nsPath, modsPath, navsPath, pluginsPath, saveDataPath string) (*NSOverlay, error) {
	n := new(NSOverlay)
	if p, err := os.MkdirTemp(dir, "ns*"); err != nil {
		return nil, fmt.Errorf("create temp dir in %q: %w", dir, err)
	} else {
		n.Path = p
	}
	if err := n.mergeTF(tfPath); err != nil {
		n.Delete()
		return nil, fmt.Errorf("merge titanfall: %w", err)
	}
	if err := n.mergeNS(nsPath); err != nil {
		n.Delete()
		return nil, fmt.Errorf("merge northstar: %w", err)
	}
	if err := n.mergeMods(modsPath); err != nil {
		n.Delete()
		return nil, fmt.Errorf("merge extra mods: %w", err)
	}
	if err := n.mergeNavs(navsPath); err != nil {
		n.Delete()
		return nil, fmt.Errorf("merge navs: %w", err)
	}
	if err := n.mergePlugins(pluginsPath); err != nil {
		n.Delete()
		return nil, fmt.Errorf("merge plugins: %w", err)
	}
	if err := n.mergeSaveData(saveDataPath); err != nil {
		n.Delete()
		return nil, fmt.Errorf("merge save data: %w", err)
	}
	return n, nil
}

// Delete attempts to delete the overlay. All files within the overlay directory
// will be removed.
func (n *NSOverlay) Delete() error {
	return os.RemoveAll(n.Path)
}

func (n *NSOverlay) Executable() string {
	return "NorthstarLauncher.exe"
}

func (n *NSOverlay) Autoexec() string {
	return filepath.Join(n.Path, "R2Northstar/mods/Northstar.CustomServers/mod/cfg/autoexec_ns_server.cfg")
}

// TODO: rewrite this with more checks and individual file symlinks

func (n *NSOverlay) mergeTF(p string) error {
	for _, x := range []string{
		"bin/x64_retail",
		"vpk",
		"r2",
		"build.txt",
		"server.dll",
	} {
		if y := filepath.Dir(x); y != "." {
			if err := os.MkdirAll(filepath.Join(n.Path, y), 0777); err != nil {
				return err
			}
		}
		if err := checkedSymlink(filepath.Join(p, x), filepath.Join(n.Path, x), false); err != nil {
			return err
		}
	}
	return nil
}

func (n *NSOverlay) mergeNS(p string) error {
	if _, err := os.Stat(filepath.Join(p, "R2Northstar/mods/Northstar.CustomServers/mod/cfg/autoexec_ns_server.cfg")); err != nil {
		return fmt.Errorf("northstar build missing server autoexec: %w", err)
	}
	for _, x := range []string{
		"bin/x64_dedi/d3d11.dll",
		"bin/x64_dedi/GFSDK_SSAO.win64.dll",
		"bin/x64_dedi/GFSDK_TXAA.win64.dll",
	} {
		if _, err := os.Stat(filepath.Join(p, x)); err != nil {
			return fmt.Errorf("northstar build missing stubs (is it 1.6 or newer?): %w", err)
		}
	}
	for _, x := range []string{
		"R2Northstar/mods/Northstar.CustomServers/mod/maps/navmesh",
		"R2Northstar/mods/Northstar.CustomServers/mod/maps/graphs",
	} {
		if _, err := os.Stat(filepath.Join(p, x)); err != nil {
			return fmt.Errorf("northstar build missing navs (is it 1.7 or newer?): %w", err)
		}
	}
	// ns wants to write into it's directory, and it also doesn't seem to work
	// properly if it's dir is symlinked...
	return filepath.Walk(p, func(path string, info fs.FileInfo, err error) error {
		r, err := filepath.Rel(p, path)
		if err != nil {
			return err
		}
		switch filepath.ToSlash(r) {
		case "R2Northstar/mods/Northstar.CustomServers/mod/cfg/autoexec_ns_server.cfg":
			return os.WriteFile(filepath.Join(n.Path, r), nil, 0666)
		case "R2Northstar/placeholder_playerdata.pdata":
			// northstar after v1.10.0 doesn't need this file anymore
			// northstar until v1.8.1 opens it read/write, so must copy
			return copyFile(path, filepath.Join(n.Path, r))
		case
			"discord_game_sdk.dll",
			"bin/x64_retail",
			"bin/x64_retail/wsock32.dll",
			"R2Northstar/plugins/DiscordRPC.dll":
			return nil
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(n.Path, r), 0777)
		} else {
			return checkedSymlink(path, filepath.Join(n.Path, r), false)
		}
	})
}

func (n *NSOverlay) mergeMods(p string) error {
	es, err := os.ReadDir(p)
	if err != nil {
		return err
	}
	for _, e := range es {
		if _, err := os.Stat(filepath.Join(n.Path, "R2Northstar/mods", e.Name())); err == nil {
			return fmt.Errorf("not allowed to override built-in mod %s", e.Name())
		}
		if err := checkedSymlink(filepath.Join(p, e.Name()), filepath.Join(n.Path, "R2Northstar/mods", e.Name()), false); err != nil {
			return err
		}
	}
	return nil
}

func (n *NSOverlay) mergeNavs(p string) error {
	// note: sorted lexically
	return filepath.Walk(p, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".ain":
			return checkedSymlink(path, filepath.Join(n.Path, "R2Northstar", "mods", "Northstar.CustomServers", "mod", "maps", "graphs", info.Name()), true)
		case ".nm":
			return checkedSymlink(path, filepath.Join(n.Path, "R2Northstar", "mods", "Northstar.CustomServers", "mod", "maps", "navmesh", info.Name()), true)
		}
		return nil
	})
}

func (n *NSOverlay) mergePlugins(p string) error {
	// note: sorted lexically
	if err := os.MkdirAll(filepath.Join(n.Path, "R2Northstar", "plugins"), 0644); err != nil {
		return err
	}
	return filepath.Walk(p, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".dll" {
			return checkedSymlink(path, filepath.Join(n.Path, "R2Northstar", "plugins", info.Name()), true)
		}
		return nil
	})
}

func (n *NSOverlay) mergeSaveData(p string) error {
	if err := checkedSymlink(p, filepath.Join(n.Path, "R2Northstar", "save_data"), true); !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func checkedSymlink(oldname, newname string, replace bool) error {
	if _, err := os.Stat(oldname); err != nil {
		return fmt.Errorf("access %q: %w", oldname, err)
	}
	if replace {
		if _, err := os.Stat(newname); err == nil {
			os.RemoveAll(newname)
		}
	}
	return os.Symlink(oldname, newname)
}

func copyFile(src, dst string) error {
	rf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer rf.Close()

	st, err := rf.Stat()
	if err != nil {
		return err
	}

	wf, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, st.Mode())
	if err != nil {
		return err
	}
	defer wf.Close()

	if _, err := io.Copy(wf, rf); err != nil {
		os.Remove(wf.Name())
		return err
	}

	if err := wf.Close(); err != nil {
		os.Remove(wf.Name())
		return err
	}
	return nil
}
