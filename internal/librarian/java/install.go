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

package java

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/tool/maven"
)

const (
	envPath  = "PATH"
	toolsDir = "java_tools"
)

// errNoToolsSpecified indicates no Java tools were provided in the configuration.
var errNoToolsSpecified = errors.New("no tools specified in configuration")

// Install installs Java tool dependencies.
// It creates two sibling directories:
// - bin/ ($LIBRARIAN_BIN/java_tools/bin) stores the generated executable wrapper scripts.
// - lib/ ($LIBRARIAN_BIN/java_tools/lib) isolates the downloaded compiled .jar/.exe files.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Maven) == 0 {
		return errNoToolsSpecified
	}
	for _, cmd := range []string{"java", "mvn"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s is not installed or not in PATH, which is required for Java tool installation: %w", cmd, err)
		}
	}
	binDir, err := getBinDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("failed to create bin directory %q: %w", binDir, err)
	}
	libDir, err := getLibDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return fmt.Errorf("failed to create lib directory %q: %w", libDir, err)
	}
	return maven.Install(ctx, tools.Maven, binDir, libDir)
}

// InstallDir returns the absolute path of the installation directory for Java tools.
func InstallDir() (string, error) {
	dir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, toolsDir))
}

// getBinDir returns the absolute path of the directory where Java tool wrapper scripts are stored.
func getBinDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(installDir, "bin"))
}

// getLibDir returns the absolute path of the directory where Java tool library files (such as .jar
// or .exe files) are stored.
func getLibDir() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(installDir, "lib"))
}

// getToolsEnv returns an environment map with the Java tools bin directory prepended to the PATH.
func getToolsEnv() (map[string]string, error) {
	binDir, err := getBinDir()
	if err != nil {
		return nil, err
	}
	return map[string]string{envPath: binDir}, nil
}
