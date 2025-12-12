package rust

import (
	"testing"

	cmdtest "github.com/googleapis/librarian/internal/command"
)

func TestGetPackageName(t *testing.T) {
	expectedPackageName := "google-cloud-accessapproval-v1"
	got, err := getPackageName("testdata/package")
	if err != nil {
		t.Fatalf("error getting package name %v", err)
	}
	if got != expectedPackageName {
		t.Errorf("want packageName %s, got %s", expectedPackageName, got)
	}
}

func TestPrepareCargoWorkspace(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	cmdtest.RequireCommand(t, "taplo")
	prepareCargoWorkspace(t.Context(), "testdata")
}

func TestFormatAndValidateCreatedLibrary(t *testing.T) {
	cmdtest.RequireCommand(t, "cargo")
	cmdtest.RequireCommand(t, "env")
	cmdtest.RequireCommand(t, "typos")
	cmdtest.RequireCommand(t, "git")
	formatAndValidateLibrary(t.Context(), "testdata/package")
}
