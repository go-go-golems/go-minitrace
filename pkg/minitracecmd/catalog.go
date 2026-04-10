package minitracecmd

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type SourceRoot struct {
	Name     string
	FS       fs.FS
	RootDir  string
	Readonly bool
}

type Catalog struct {
	Commands []*MinitraceCommand
	ByPath   map[string]*MinitraceCommand
	ByName   map[string]*MinitraceCommand
}

func LoadCatalog(roots []SourceRoot) (*Catalog, error) {
	compiler := &Compiler{}
	catalog := &Catalog{
		Commands: []*MinitraceCommand{},
		ByPath:   map[string]*MinitraceCommand{},
		ByName:   map[string]*MinitraceCommand{},
	}

	for _, root := range roots {
		rootDir := root.RootDir
		if rootDir == "" {
			rootDir = "."
		}

		err := fs.WalkDir(root.FS, rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			kind := DetectSourceKind(path)
			if kind == SourceUnknown {
				return nil
			}

			contents, err := fs.ReadFile(root.FS, path)
			if err != nil {
				return err
			}

			var spec *MinitraceCommandSpec
			switch kind {
			case SourceSQLCommand:
				if !LooksLikeSqletonSQLCommand(contents) {
					return nil
				}
				spec, err = ParseSQLCommandSpec(path, contents)
			case SourceYAMLAlias:
				spec, err = ParseAliasSpec(path, contents)
			case SourceUnknown:
				return nil
			}
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			folder := filepath.ToSlash(filepath.Dir(rel))
			if folder == "." {
				folder = ""
			}

			cmd, err := compiler.Compile(spec, CompileOptions{
				Folder:     folder,
				Path:       rel,
				SourceRoot: root.Name,
				SourcePath: path,
				Readonly:   root.Readonly,
			})
			if err != nil {
				return err
			}

			if _, exists := catalog.ByPath[cmd.Path]; exists {
				return nil // first root wins
			}
			catalog.ByPath[cmd.Path] = cmd
			catalog.Commands = append(catalog.Commands, cmd)
			if cmd.Kind == MinitraceCommandVerb {
				if _, exists := catalog.ByName[cmd.Name]; !exists {
					catalog.ByName[cmd.Name] = cmd
				}
			}

			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "load catalog root %q", root.Name)
		}
	}

	sort.Slice(catalog.Commands, func(i, j int) bool {
		return strings.Compare(catalog.Commands[i].Path, catalog.Commands[j].Path) < 0
	})

	if err := resolveAliases(catalog); err != nil {
		return nil, err
	}

	return catalog, nil
}

func resolveAliases(catalog *Catalog) error {
	for _, cmd := range catalog.Commands {
		if cmd.Kind != MinitraceCommandAlias {
			continue
		}
		if _, ok := catalog.ByName[cmd.AliasFor]; !ok {
			return errors.Wrapf(ErrAliasTargetNotFound, "alias %q targets missing command %q", cmd.Name, cmd.AliasFor)
		}
	}
	return nil
}
