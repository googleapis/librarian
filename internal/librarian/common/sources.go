package common

import (
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Sources contains the directory paths for source repositories used by
// sidekick.
type Sources struct {
	Conformance string
	Discovery   string
	Googleapis  string
	ProtobufSrc string
	Showcase    string
}

func AddLibraryRoots(library *config.Library, sources *Sources) map[string]string {
	source := make(map[string]string)
	if library.Dart == nil {
		library.Dart = &config.DartPackage{}
	}

	if len(library.Roots) == 0 && sources.Googleapis != "" {
		// Default to googleapis if no roots are specified.
		source["googleapis-root"] = sources.Googleapis
		source["roots"] = "googleapis"
	} else {
		source["roots"] = strings.Join(library.Roots, ",")
		rootMap := map[string]struct {
			path string
			key  string
		}{
			"googleapis":   {path: sources.Googleapis, key: "googleapis-root"},
			"discovery":    {path: sources.Discovery, key: "discovery-root"},
			"showcase":     {path: sources.Showcase, key: "showcase-root"},
			"protobuf-src": {path: sources.ProtobufSrc, key: "protobuf-src-root"},
			"conformance":  {path: sources.Conformance, key: "conformance-root"},
		}
		for _, root := range library.Roots {
			if r, ok := rootMap[root]; ok && r.path != "" {
				source[r.key] = r.path
			}
		}
	}

	return source
}
