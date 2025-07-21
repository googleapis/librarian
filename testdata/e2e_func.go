package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	inputDir  = "input"
	librarian = "librarian"
	libraryID = "library-id"
	outputDir = "output"
	source    = "source"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatal(errors.New("no command-line arguments provided"))
	}

	log.Print("received command: ", os.Args[1:])
	switch os.Args[1] {
	case "generate":
		if err := validateGenerate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unrecognized command: ", os.Args[1])
	}
}

func validateGenerate(args []string) error {
	for _, arg := range args {
		arg, _ := strings.CutPrefix(arg, "--")
		strs := strings.Split(arg, "=")
		switch strs[0] {
		case librarian:
			if err := validateLibrarianDir(strs[1]); err != nil {
				return err
			}
		case inputDir:
			return nil
		case outputDir:
			return nil
		case source:
			return nil
		default:
			log.Fatal("unrecognized option: ", arg)
		}
	}

	return nil
}

func validateLibrarianDir(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "generate-request.json")); err != nil {
		return err
	}

	return nil
}
