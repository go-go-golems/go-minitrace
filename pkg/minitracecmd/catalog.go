package minitracecmd

import (
	"fmt"
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
	Commands    []*MinitraceCommand
	ByPath      map[string]*MinitraceCommand
	ByName      map[string]*MinitraceCommand
	SourceRoots map[string]SourceRoot
}

func LoadCatalog(roots []SourceRoot) (*Catalog, error) {
	compiler := &Compiler{}
	catalog := &Catalog{
		Commands:    []*MinitraceCommand{},
		ByPath:      map[string]*MinitraceCommand{},
		ByName:      map[string]*MinitraceCommand{},
		SourceRoots: map[string]SourceRoot{},
	}
	seenSourcePaths := map[string]struct{}{}
	seenCommandPaths := map[string]string{}

	for _, root := range roots {
		rootKey := uniqueSourceRootKey(root.Name, catalog.SourceRoots)
		catalog.SourceRoots[rootKey] = root
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

			rel, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if _, ok := seenSourcePaths[rel]; ok {
				return nil // first root wins for a source file path
			}

			contents, err := fs.ReadFile(root.FS, path)
			if err != nil {
				return err
			}

			parsed, err := parseSourceSpecs(path, rel, kind, contents)
			if err != nil {
				return err
			}
			if kind == SourceJSCommand {
				parsed = expandCollapsedSelfNamedJSPathIfNestedSources(root.FS, path, rel, parsed)
			}
			if len(parsed) == 0 {
				return nil
			}
			seenSourcePaths[rel] = struct{}{}

			for _, entry := range parsed {
				if entry.Spec == nil {
					continue
				}
				folder := filepath.ToSlash(filepath.Dir(entry.Path))
				if folder == "." {
					folder = ""
				}

				cmd, err := compiler.Compile(entry.Spec, CompileOptions{
					Folder:     folder,
					Path:       entry.Path,
					SourceRoot: rootKey,
					SourcePath: path,
					Readonly:   root.Readonly,
				})
				if err != nil {
					return err
				}

				logicalPath := commandLogicalPath(cmd)
				if existingSource, exists := seenCommandPaths[logicalPath]; exists {
					return errors.Wrapf(ErrDuplicateCommandPath, "%s already defined by %s", logicalPath, existingSource)
				}
				seenCommandPaths[logicalPath] = cmd.SourcePath

				if _, exists := catalog.ByPath[cmd.Path]; exists {
					return errors.Wrapf(ErrDuplicateCommandPath, "%s already defined", cmd.Path)
				}
				catalog.ByPath[cmd.Path] = cmd
				catalog.Commands = append(catalog.Commands, cmd)
				if cmd.Kind == MinitraceCommandVerb {
					if _, exists := catalog.ByName[cmd.Name]; !exists {
						catalog.ByName[cmd.Name] = cmd
					}
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

func parseSourceSpecs(sourcePath string, rel string, kind SourceKind, contents []byte) ([]ParsedCommandSpec, error) {
	switch kind {
	case SourceUnknown:
		return nil, nil
	case SourceSQLCommand:
		if !LooksLikeSqletonSQLCommand(contents) {
			return nil, nil
		}
		spec, err := ParseSQLCommandSpec(sourcePath, contents)
		if err != nil {
			return nil, err
		}
		return []ParsedCommandSpec{{Spec: spec, Path: rel}}, nil
	case SourceJSCommand:
		return ParseJSCommandSpecs(rel, contents)
	case SourceYAMLAlias:
		spec, err := ParseAliasSpec(sourcePath, contents)
		if err != nil {
			return nil, err
		}
		return []ParsedCommandSpec{{Spec: spec, Path: rel}}, nil
	}

	return nil, nil
}

func commandLogicalPath(cmd *MinitraceCommand) string {
	if cmd == nil {
		return ""
	}
	if strings.TrimSpace(cmd.Folder) == "" {
		return cmd.Name
	}
	return cmd.Folder + "/" + cmd.Name
}

func expandCollapsedSelfNamedJSPathIfNestedSources(fsys fs.FS, sourcePath, rel string, parsed []ParsedCommandSpec) []ParsedCommandSpec {
	if len(parsed) != 1 || parsed[0].Spec == nil || parsed[0].Spec.Runtime != CommandRuntimeJS {
		return parsed
	}

	spec := parsed[0].Spec
	collapsedPath := jsCommandPath(rel, spec.Name, true)
	expandedPath := jsCommandPath(rel, spec.Name, false)
	if collapsedPath == expandedPath || parsed[0].Path != collapsedPath {
		return parsed
	}

	siblingDir := strings.TrimSuffix(filepath.ToSlash(sourcePath), filepath.Ext(sourcePath))
	if !hasNestedCommandSources(fsys, siblingDir) {
		return parsed
	}

	ret := append([]ParsedCommandSpec(nil), parsed...)
	ret[0].Path = expandedPath
	return ret
}

func hasNestedCommandSources(fsys fs.FS, dir string) bool {
	found := false
	_ = fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if DetectSourceKind(path) != SourceUnknown {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

func uniqueSourceRootKey(name string, roots map[string]SourceRoot) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "root"
	}
	if _, exists := roots[base]; !exists {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s#%d", base, i)
		if _, exists := roots[candidate]; !exists {
			return candidate
		}
	}
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
