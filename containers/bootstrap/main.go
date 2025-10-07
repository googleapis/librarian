package bootstrap
import (
    "fmt"
    "os"
    "context"
)

// GenerateFlags is the flags in Librarian's container contract for the
// generate command.
// https://github.com/googleapis/librarian/blob/main/doc/language-onboarding.md#generate
type GenerateFlags struct {
	Librarian string
	Input string
	Output string
	Source string
}



func LanguageContainerMain(generateFunc func(ctx context.Context, generateFlags *GenerateFlags)) {
	fmt.Println("Arguments: %v", os.Args)
	generateFlags := GenerateFlags{}
	generateFunc(nil, &generateFlags)
}
