// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package filesystem provides generic filesystem operations.
package filesystem

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MoveAndMerge moves entries from sourceDir to targetDir.
// If an entry is a directory and already exists in targetDir, it merges the contents recursively.
// Otherwise, it renames the entry.
func MoveAndMerge(sourceDir, targetDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		oldPath := filepath.Join(sourceDir, entry.Name())
		newPath := filepath.Join(targetDir, entry.Name())
		if entry.IsDir() {
			if _, err := os.Stat(newPath); err == nil {
				// Destination exists, merge contents.
				if err := MoveAndMerge(oldPath, newPath); err != nil {
					return err
				}
				continue
			}
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}
	return nil
}

// CopyFile copies a file from src to dest.
func CopyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// Unzip unzips the src archive into dest directory.
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, copyErr := io.Copy(outFile, rc)
		srcCloseErr := rc.Close()
		dstCloseErr := outFile.Close()

		if copyErr != nil {
			return copyErr
		}
		if srcCloseErr != nil {
			return srcCloseErr
		}
		if dstCloseErr != nil {
			return dstCloseErr
		}
	}
	return nil
}
