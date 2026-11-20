package minitracecmd

import (
	"io/fs"
	"path"
)

// RuntimeSourceFS omits diagnosed sources from runtime scanning as well as discovery.
// A command cannot be rescued by discovery alone if ScanFS re-parses a bad sibling.
func (c *Catalog) RuntimeSourceFS(rootKey string) fs.FS {
	root := c.SourceRoots[rootKey]
	excluded := map[string]bool{}
	for _, d := range c.Diagnostics {
		if d.SourceRoot == rootKey {
			excluded[d.SourcePath] = true
		}
	}
	return diagnosedFS{FS: root.FS, excluded: excluded}
}

type diagnosedFS struct {
	fs.FS
	excluded map[string]bool
}

func (f diagnosedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.FS, name)
	if err != nil {
		return nil, err
	}
	result := make([]fs.DirEntry, 0, len(entries))
	for _, e := range entries {
		if !f.excluded[path.Join(name, e.Name())] {
			result = append(result, e)
		}
	}
	return result, nil
}
