package proto

import (
	"os"
	"path/filepath"
	"slices"
)

var (
	// nonRecursivePaths is a set of paths where proto gathering should not be recursive.
	nonRecursivePaths = map[string]bool{
		"google/api":   true,
		"google/cloud": true,
		"google/rpc":   true,
	}
)

// Gather returns a sorted list of proto files in the given root directory,
// ensuring that subpackage protos (e.g., in a "schema" directory) are included
// in the generation.
//
// recursion is disabled for certain base paths in nonRecursivePaths.
func Gather(root, relPath string) ([]string, error) {
	var protos []string
	recursive := !nonRecursivePaths[filepath.ToSlash(relPath)]
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && filepath.Ext(path) == ".proto" {
			protos = append(protos, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(protos)
	return protos, nil
}
