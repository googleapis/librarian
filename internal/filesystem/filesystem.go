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
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
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
				// Remove the now-empty source directory after successful merge.
				if err := os.Remove(oldPath); err != nil {
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
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// Unzip unzips the src archive into dest directory using the system unzip command.
func Unzip(ctx context.Context, src, dest string) error {
	return command.Run(ctx, "unzip", "-q", "-o", src, "-d", dest)
}
