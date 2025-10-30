package pom

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

var (
	versionRegex = regexp.MustCompile(`(<version>)([^<]+)(\s*<!-- \{x-version-update:([^:]+):current\} -->\s*</version>)`)
)

// UpdateVersions updates the versions of all pom.xml files in a given directory.
func UpdateVersions(path, libraryID, version string) error {
	pomFiles, err := findPomFiles(path)
	if err != nil {
		return fmt.Errorf("failed to find pom files: %w", err)
	}
	for _, pomFile := range pomFiles {
		if err := updateVersion(pomFile, libraryID, version); err != nil {
			return fmt.Errorf("failed to update version in %s: %w", pomFile, err)
		}
	}
	return nil
}

func updateVersion(path, libraryID, version string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	newContent := versionRegex.ReplaceAllStringFunc(string(content), func(s string) string {
		matches := versionRegex.FindStringSubmatch(s)
		if len(matches) > 4 && matches[4] == libraryID {
			// matches[1] is "<version>"
			// matches[2] is the old version
			// matches[3] is " <!-- {x-version-update:libraryID:current} --> </version>"
			// matches[4] is libraryID
			return fmt.Sprintf("%s%s%s", matches[1], version, matches[3])
		}
		return s
	})

	if newContent == string(content) {
		return nil // No change made
	}

	if err := ioutil.WriteFile(path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func findPomFiles(path string) ([]string, error) {
	var pomFiles []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "pom.xml" {
			pomFiles = append(pomFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk path: %w", err)
	}
	return pomFiles, nil
}
