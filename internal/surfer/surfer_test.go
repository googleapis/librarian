package surfer

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		err  error
	}{
		{
			name: "valid command",
			args: []string{"generate",
				"--config", "gcloud/testdata/parallelstore/gcloud.yaml",
				"--out", "gcloud/testdata/parallelstore/surface",
			},
		},
		{
			name: "missing config flag",
			args: []string{"generate"},
			err:  errMissingConfigFlag,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Run(t.Context(), test.args); err != nil {
				// TODO(https://github.com/googleapis/librarian/issues/2817):
				// return once the generate functionality has been implemented
				if strings.Contains(err.Error(), "failed to create API model") {
					return
				}
				if err != test.err {
					t.Fatal(err)
				}
			}
		})
	}
}
