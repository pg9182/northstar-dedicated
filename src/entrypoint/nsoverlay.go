package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type NSOverlay struct {
	Path string
}

func MergeOverlay(dir, tfPath, nsPath, modsPath, navsPath string) (*NSOverlay, error) {
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
		"build.txt",
		"server.dll",
	} {
		if y := filepath.Dir(x); y != "." {
			if err := os.MkdirAll(filepath.Join(n.Path, y), 0777); err != nil {
				return err
			}
		}
		if err := checkedSymlink(filepath.Join(p, x), filepath.Join(n.Path, x)); err != nil {
			return err
		}
	}
	{
		if err := os.MkdirAll(filepath.Join(n.Path, "r2"), 0777); err != nil {
			return err
		}
		es, err := os.ReadDir(filepath.Join(p, "r2"))
		if err != nil {
			return err
		}
		for _, e := range es {
			switch e.Name() {
			case "maps":
				if err := os.MkdirAll(filepath.Join(n.Path, "r2", e.Name()), 0777); err != nil {
					return err
				}
			default:
				if err := checkedSymlink(filepath.Join(p, "r2", e.Name()), filepath.Join(n.Path, "r2", e.Name())); err != nil {
					return err
				}
			}
		}
	}
	// the r2/maps dir isn't required for dedicated servers, but we'll include
	// stuff from there if it's present
	if es, err := os.ReadDir(filepath.Join(p, "r2", "maps")); err == nil {
		for _, e := range es {
			switch e.Name() {
			case "navmesh", "graphs":
				if err := os.MkdirAll(filepath.Join(n.Path, "r2", "maps", e.Name()), 0777); err != nil {
					return err
				}
				es1, err := os.ReadDir(filepath.Join(p, "r2", "maps", e.Name()))
				if err != nil {
					return err
				}
				for _, e1 := range es1 {
					if err := checkedSymlink(filepath.Join(p, "r2", "maps", e.Name(), e1.Name()), filepath.Join(n.Path, "r2", "maps", e.Name(), e1.Name())); err != nil {
						return err
					}
				}
			default:
				if err := checkedSymlink(filepath.Join(p, "r2", "maps", e.Name()), filepath.Join(n.Path, "r2", "maps", e.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (n *NSOverlay) mergeNS(p string) error {
	for _, x := range []string{
		"bin/x64_dedi/d3d11.dll",
		"bin/x64_dedi/GFSDK_SSAO.win64.dll",
		"bin/x64_dedi/GFSDK_TXAA.win64.dll",
	} {
		if _, err := os.Stat(filepath.Join(p, x)); err != nil {
			return fmt.Errorf("northstar build missing stubs (is it 1.6 or newer?): %w", err)
		}
	}
	// ns wants to write into it's directory, and it also doesn't seem to work
	// properly if it's dir is symlinked...
	return filepath.Walk(p, func(path string, info fs.FileInfo, err error) error {
		r, err := filepath.Rel(p, path)
		if err != nil {
			return err
		}
		if info.Name() == "autoexec_ns_server.cfg" {
			return os.WriteFile(filepath.Join(n.Path, r), nil, 0666)
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(n.Path, r), 0777)
		} else {
			return checkedSymlink(path, filepath.Join(n.Path, r))
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
		if err := checkedSymlink(filepath.Join(p, e.Name()), filepath.Join(n.Path, "R2Northstar/mods", e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (n *NSOverlay) mergeNavs(p string) error {
	// note: sorted lexically
	for _, x := range []string{"navmesh", "graphs"} {
		if err := os.MkdirAll(filepath.Join(n.Path, "r2", "maps", x), 0777); err != nil {
			return err
		}
	}
	filepath.Walk(p, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".ain":
			return checkedSymlink(path, filepath.Join(n.Path, "r2", "maps", "graphs", info.Name()))
		case ".nm":
			return checkedSymlink(path, filepath.Join(n.Path, "r2", "maps", "navmesh", info.Name()))
		}
		return nil
	})
	return nil
}

func checkedSymlink(oldname, newname string) error {
	if _, err := os.Stat(oldname); err != nil {
		return fmt.Errorf("access %q: %w", oldname, err)
	}
	return os.Symlink(oldname, newname)
}
